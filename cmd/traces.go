package cmd

import (
	"fmt"
	"strconv"

	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var tracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Transaction traces",
}

var tracesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List traces for an endpoint",
	Run:   runTracesList,
}

var tracesShowCmd = &cobra.Command{
	Use:   "show <trace-id>",
	Short: "Show trace detail with span tree",
	Args:  cobra.ExactArgs(1),
	Run:   runTracesShow,
}

var tracesEndpointFlag string

func init() {
	tracesListCmd.Flags().StringVar(&tracesEndpointFlag, "endpoint", "", "URL-encoded endpoint ID")
	_ = tracesListCmd.MarkFlagRequired("endpoint")
	tracesCmd.AddCommand(tracesListCmd, tracesShowCmd)
	rootCmd.AddCommand(tracesCmd)
}

func runTracesList(cmd *cobra.Command, args []string) {
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

	traces, err := client.ListTraces(id, tracesEndpointFlag, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(traces) {
		return
	}

	total := len(traces)
	limit, _ := applyLimit(total)

	headers := []string{"ID", "Time", "Duration", "Memory", "Endpoint", "URI"}
	rows := make([][]string, limit)
	for i := 0; i < limit; i++ {
		t := traces[i]
		rows[i] = []string{
			strconv.Itoa(t.ID),
			output.FormatRelativeTime(t.Time),
			output.FormatSeconds(t.TotalCallTime),
			output.FormatBytes(t.MemDelta),
			t.MetricName,
			t.URI,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
}

func runTracesShow(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	traceID, err := strconv.Atoi(args[0])
	if err != nil {
		exitError("invalid trace ID: " + args[0])
	}

	trace, err := client.GetTrace(id, traceID)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(trace) {
		return
	}

	fmt.Println(output.RenderSpanTree(trace))
}
