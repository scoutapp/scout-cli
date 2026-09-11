package cmd

import (
	"fmt"
	"strconv"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var tracesCmd = &cobra.Command{
	Use:   "traces",
	Short: "Transaction traces",
}

var tracesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List traces for an endpoint or background job",
	Long: `List traces for an endpoint (--endpoint) or a background job (--job).

Exactly one of --endpoint or --job is required. Traces are the slowest
recorded within the timeframe (max 100, within the last 7 days).`,
	Example: `  # Traces for an endpoint, by the Base64 endpoint ID in the
  # "link" field of 'scout endpoints list --json'
  scout traces list --endpoint YXBpL21ldHJpY3Mvc2hvdw== --app 6

  # Traces for a background job, by the "job_id" field of
  # 'scout jobs list --json'
  scout traces list --job ZGVmYXVsdC9NeVdvcmtlcg== --app 6

  # The same job by its "queue/JobName" full name
  scout traces list --job default/MyWorker --app 6 --from 7d`,
	Run: runTracesList,
}

var tracesShowCmd = &cobra.Command{
	Use:   "show <trace-id>",
	Short: "Show trace detail with span tree",
	Long: `Show trace detail with span tree.

Only endpoint (web request) traces are supported; the API does not yet
expose detail for background job traces.`,
	Args: cobra.ExactArgs(1),
	Run:  runTracesShow,
}

var (
	tracesEndpointFlag string
	tracesJobFlag      string
)

func init() {
	tracesListCmd.Flags().StringVar(&tracesEndpointFlag, "endpoint", "", "URL-encoded endpoint ID")
	tracesListCmd.Flags().StringVar(&tracesJobFlag, "job", "", "Job ID or queue/JobName full name (lists traces for a background job)")
	tracesListCmd.MarkFlagsOneRequired("endpoint", "job")
	tracesListCmd.MarkFlagsMutuallyExclusive("endpoint", "job")
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

	if tracesJobFlag != "" {
		runJobTracesList(client, id, from, to)
		return
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
			output.FormatMs(t.TotalCallTime),
			output.FormatMB(t.MemDelta),
			t.MetricName,
			t.URI,
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(limit, total)
}

func runJobTracesList(client *api.Client, id int, from, to string) {
	traces, err := client.ListJobTraces(id, resolveJobID(tracesJobFlag), from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(traces) {
		return
	}

	total := len(traces)
	limit, _ := applyLimit(total)

	headers := []string{"ID", "Time", "Duration", "Job", "Queue"}
	rows := make([][]string, limit)
	for i := 0; i < limit; i++ {
		t := traces[i]
		rows[i] = []string{
			strconv.Itoa(t.ID),
			output.FormatRelativeTime(t.Time),
			output.FormatMs(t.Duration),
			t.Name,
			t.Queue,
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
