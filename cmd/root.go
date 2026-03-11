package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
	Use:     "scout",
	Short:   "Scout APM CLI — monitor application performance from the terminal",
	Long:    "A command-line interface for Scout APM. View apps, metrics, endpoints, traces, errors, and insights.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Auto-enable JSON when piped
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
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	rootCmd.PersistentFlags().IntVar(&appID, "app", 0, "Application ID")
	rootCmd.PersistentFlags().StringVar(&fromFlag, "from", "", "Start time (relative: 1h, 7d, 30m or ISO 8601)")
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

func resolveTimeframe() (string, string, error) {
	return timeutil.ResolveTimeframe(fromFlag, toFlag)
}

func outputJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]interface{}{"data": data})
}

func applyLimit(total int) (limit int, truncated bool) {
	if limitFlag > 0 && limitFlag < total {
		return limitFlag, true
	}
	return total, false
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
