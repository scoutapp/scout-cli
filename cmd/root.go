package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/scoutapm/scout-cli/internal/api"
	"github.com/scoutapm/scout-cli/internal/config"
	"github.com/scoutapm/scout-cli/internal/timeutil"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonOutput bool
	appID      int
	fromFlag   string
	toFlag     string
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "Scout APM CLI — monitor application performance from the terminal",
	Long:  "A command-line interface for Scout APM. View apps, metrics, endpoints, traces, errors, and insights.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Auto-enable JSON when piped
		if !jsonOutput && !term.IsTerminal(int(os.Stdout.Fd())) {
			jsonOutput = true
		}
		if noColor {
			os.Setenv("NO_COLOR", "1")
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
	enc.Encode(map[string]interface{}{"data": data})
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
