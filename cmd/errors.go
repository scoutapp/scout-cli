package cmd

import (
	"fmt"
	"strconv"

	"github.com/scoutapm/scout-cli/internal/output"
	"github.com/spf13/cobra"
)

var errorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Error groups and occurrences",
}

var errorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List error groups",
	Run:   runErrorsList,
}

var errorsShowCmd = &cobra.Command{
	Use:   "show <error-id>",
	Short: "Show error group details",
	Args:  cobra.ExactArgs(1),
	Run:   runErrorsShow,
}

var errorsOccurrencesCmd = &cobra.Command{
	Use:   "occurrences <error-id>",
	Short: "List individual error occurrences",
	Args:  cobra.ExactArgs(1),
	Run:   runErrorsOccurrences,
}

func init() {
	errorsCmd.AddCommand(errorsListCmd, errorsShowCmd, errorsOccurrencesCmd)
	rootCmd.AddCommand(errorsCmd)
}

func runErrorsList(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	groups, err := client.ListErrorGroups(id, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(groups)
		return
	}

	headers := []string{"ID", "Name", "Count", "Status", "Last Seen"}
	rows := make([][]string, len(groups))
	for i, g := range groups {
		status := output.StatusColor(g.Status).Render(g.Status)
		rows[i] = []string{
			strconv.Itoa(g.ID),
			g.Name,
			strconv.Itoa(g.ErrorsCount),
			status,
			output.FormatRelativeTime(g.LastErrorAt),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
}

func runErrorsShow(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	errorID, err := strconv.Atoi(args[0])
	if err != nil {
		exitError("invalid error ID: " + args[0])
	}

	group, err := client.GetErrorGroup(id, errorID)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(group)
		return
	}

	fmt.Println(output.HeaderStyle.Render(group.Name))
	fmt.Printf("  Message:  %s\n", group.Message)
	fmt.Printf("  Status:   %s\n", output.StatusColor(group.Status).Render(group.Status))
	fmt.Printf("  Count:    %d\n", group.ErrorsCount)
	fmt.Printf("  Last:     %s\n", output.FormatRelativeTime(group.LastErrorAt))
	if group.RequestURI != "" {
		fmt.Printf("  URI:      %s\n", group.RequestURI)
	}
	if group.AppEnvironment != "" {
		fmt.Printf("  Env:      %s\n", group.AppEnvironment)
	}

	if group.LatestError != nil {
		fmt.Println()
		fmt.Println(output.BoldStyle.Render("Latest Occurrence"))
		le := group.LatestError
		fmt.Printf("  Time:     %s\n", output.FormatRelativeTime(le.CreatedAt))
		if le.Location != "" {
			fmt.Printf("  Location: %s\n", le.Location)
		}
		if len(le.Trace) > 0 {
			fmt.Println()
			fmt.Println(output.DimStyle.Render("  Backtrace:"))
			limit := 10
			if len(le.Trace) < limit {
				limit = len(le.Trace)
			}
			for _, line := range le.Trace[:limit] {
				fmt.Printf("    %s\n", output.DimStyle.Render(line))
			}
			if len(le.Trace) > 10 {
				fmt.Printf("    %s\n", output.DimStyle.Render(fmt.Sprintf("... and %d more lines", len(le.Trace)-10)))
			}
		}
	}
}

func runErrorsOccurrences(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	errorID, err := strconv.Atoi(args[0])
	if err != nil {
		exitError("invalid error ID: " + args[0])
	}

	occurrences, err := client.ListErrorOccurrences(id, errorID)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(occurrences)
		return
	}

	headers := []string{"ID", "Time", "Location", "URI", "Message"}
	rows := make([][]string, len(occurrences))
	for i, o := range occurrences {
		msg := o.Message
		if len(msg) > 80 {
			msg = msg[:77] + "..."
		}
		rows[i] = []string{
			strconv.Itoa(o.ID),
			output.FormatRelativeTime(o.CreatedAt),
			o.Location,
			o.RequestURI,
			msg,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
}
