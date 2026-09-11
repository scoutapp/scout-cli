package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/config"
	"github.com/scoutapm/scout/internal/output"
	"github.com/scoutapm/scout/internal/timeutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Version is set at build time via ldflags.
	Version = "dev"

	jsonOutput bool
	appID      int
	fromFlag   string
	toFlag     string
	noColor    bool
	limitFlag  int
)

var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "Scout APM CLI — monitor application performance from the terminal",
	Long: "A command-line interface for Scout APM. View apps, metrics, endpoints, background jobs, traces, errors, anomalies, insights, usage, and billing.\n\n" +
		"When piped, output defaults to JSON. Use --json to force JSON in a terminal.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Auto-enable JSON when piped, so scripts and agents get structured output.
		if !jsonOutput && !term.IsTerminal(int(os.Stdout.Fd())) {
			jsonOutput = true
		}
		if noColor {
			_ = os.Setenv("NO_COLOR", "1")
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON (auto-enabled when piped)")
	rootCmd.PersistentFlags().IntVar(&appID, "app", 0, "Application ID")
	rootCmd.PersistentFlags().StringVar(&fromFlag, "from", "", "Start time (relative: 30m, 1h, 7d, 2w or ISO 8601)")
	rootCmd.PersistentFlags().StringVar(&toFlag, "to", "", "End time (relative or ISO 8601, default: now)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colors")
	rootCmd.PersistentFlags().IntVarP(&limitFlag, "limit", "n", 0, "Max number of results to show (0 = no limit)")
}

func getClient() (*api.Client, error) {
	key := config.GetAPIKey()
	if key == "" {
		return nil, fmt.Errorf("not authenticated — run 'scout auth login' first")
	}
	return api.NewClient(config.GetAPIURL(), key), nil
}

func requireAppID() (int, error) {
	if err := validateAppIDFlag(appID, appIDFlagPassed()); err != nil {
		return 0, err
	}
	if appID > 0 {
		return appID, nil
	}
	cfg, err := config.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read config: %w", err)
	}
	if cfg.DefaultAppID > 0 {
		return cfg.DefaultAppID, nil
	}
	return 0, fmt.Errorf("no app specified — use --app flag or set default_app_id in config")
}

// appIDFilter returns the app id for commands that treat --app as an optional
// filter rather than a requirement. Returns 0 when --app was not passed.
func appIDFilter() (int, error) {
	passed := appIDFlagPassed()
	if err := validateAppIDFlag(appID, passed); err != nil {
		return 0, err
	}
	if !passed {
		return 0, nil
	}
	return appID, nil
}

func appIDFlagPassed() bool {
	return rootCmd.PersistentFlags().Changed("app")
}

// validateAppIDFlag rejects a --app that was passed with a value that can
// never be an app id, so it isn't mistaken for the flag being absent.
func validateAppIDFlag(value int, passed bool) error {
	if passed && value <= 0 {
		return fmt.Errorf("invalid --app value: %d — app ids are positive integers", value)
	}
	return nil
}

// decodeBase64ID decodes a Base64 URL-safe id (as used for endpoint and job
// ids, matching Ruby's Base64.urlsafe_encode64) back to the name it encodes.
// Returns false if the value is not a decodable id.
func decodeBase64ID(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	b, err := base64.URLEncoding.DecodeString(id)
	if err != nil {
		b, err = base64.RawURLEncoding.DecodeString(id)
		if err != nil {
			return "", false
		}
	}
	if len(b) == 0 || !utf8.Valid(b) {
		return "", false
	}
	return string(b), true
}

// requireFlagValue rejects a blank value for a flag that needs one. Cobra's
// required-flag check is satisfied by an empty string, which would otherwise
// reach the server and come back as an opaque error.
func requireFlagValue(name, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("--%s requires a value", name)
	}
	return trimmed, nil
}

func resolveTimeframe() (string, string, error) {
	return timeutil.ResolveTimeframe(fromFlag, toFlag)
}

func outputJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]interface{}{"data": data})
}

// structuredOutput returns true and outputs the data if --json (or piped) is
// active. Returns false if human-readable output should be used.
func structuredOutput(data interface{}) bool {
	if jsonOutput {
		outputJSON(data)
		return true
	}
	return false
}

// limitSlice applies -n/--limit to a list of results, returning the rows to
// show and the total before limiting. Callers limit before handing results to
// structuredOutput so -n applies to --json output too, and report the
// untruncated total via printTruncated.
func limitSlice[T any](items []T) (shown []T, total int) {
	total = len(items)
	if limitFlag > 0 && limitFlag < total {
		return items[:limitFlag], total
	}
	return items, total
}

func printTruncated(shown, total int) {
	if shown < total {
		fmt.Printf("\n%s\n", output.DimStyle.Render(
			fmt.Sprintf("Showing %d of %d results. Use -n to adjust.", shown, total)))
	}
}

func exitError(msg string) {
	fmt.Fprintln(os.Stderr, "Error: "+msg)
	os.Exit(1)
}

func exitAuthError() {
	fmt.Fprintln(os.Stderr, "Error: Authentication failed — check your API key")
	os.Exit(2)
}

func handleAPIError(err error) {
	if apiErr, ok := err.(*api.APIError); ok {
		if apiErr.StatusCode == 403 {
			exitAuthError()
		}
		exitError(apiErr.Error())
	}
	exitError(err.Error())
}
