package cmd

import (
	"fmt"
	"strconv"

	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "Manage applications",
}

var appsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all applications",
	Run:   runAppsList,
}

var appsShowCmd = &cobra.Command{
	Use:   "show <app-id>",
	Short: "Show application details",
	Args:  cobra.ExactArgs(1),
	Run:   runAppsShow,
}

func init() {
	appsCmd.AddCommand(appsListCmd, appsShowCmd)
	rootCmd.AddCommand(appsCmd)
}

func runAppsList(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	apps, err := client.ListApps()
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(apps)
		return
	}

	total := len(apps)
	limit, _ := applyLimit(total)

	headers := []string{"ID", "Name", "Last Reported"}
	rows := make([][]string, limit)
	for i := 0; i < limit; i++ {
		app := apps[i]
		lastReported := ""
		if app.LastReportedAt != "" {
			lastReported = output.FormatRelativeTime(app.LastReportedAt)
		}
		rows[i] = []string{
			strconv.Itoa(app.ID),
			app.Name,
			lastReported,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
}

func runAppsShow(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		exitError("invalid app ID: " + args[0])
	}

	app, err := client.GetApp(id)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(app)
		return
	}

	fmt.Println(output.HeaderStyle.Render(app.Name))
	fmt.Printf("  ID: %d\n", app.ID)
	if app.LastReportedAt != "" {
		fmt.Printf("  Last Reported: %s\n", output.FormatRelativeTime(app.LastReportedAt))
	}
}
