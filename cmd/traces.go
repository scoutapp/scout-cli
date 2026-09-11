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

	if cmd.Flags().Changed("job") {
		jobID, err := resolveTracesJobID()
		if err != nil {
			exitError(err.Error())
		}
		from, to, err := resolveTimeframe()
		if err != nil {
			exitError(err.Error())
		}
		runJobTracesList(client, id, jobID, from, to)
		return
	}

	endpoint, err := requireFlagValue("endpoint", tracesEndpointFlag)
	if err != nil {
		exitError(err.Error())
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	traces, err := client.ListTraces(id, endpoint, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	traces, total := limitSlice(traces)

	if structuredOutput(traces) {
		return
	}

	headers := []string{"ID", "Time", "Duration", "Memory", "Endpoint", "URI"}
	rows := make([][]string, len(traces))
	for i := range traces {
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
	printTruncated(len(traces), total)
}

// resolveTracesJobID validates --job and returns the encoded job id.
func resolveTracesJobID() (string, error) {
	job, err := requireFlagValue("job", tracesJobFlag)
	if err != nil {
		return "", err
	}
	return resolveJobID(job)
}

func runJobTracesList(client *api.Client, id int, jobID, from, to string) {
	traces, err := client.ListJobTraces(id, jobID, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	traces, total := limitSlice(traces)

	if structuredOutput(traces) {
		return
	}

	headers := []string{"ID", "Time", "Duration", "Job", "Queue"}
	rows := make([][]string, len(traces))
	for i := range traces {
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
	printTruncated(len(traces), total)
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
		handleAPIError(traceShowError(err))
		return
	}

	if structuredOutput(trace) {
		return
	}

	fmt.Println(output.RenderSpanTree(trace))
}

// traceShowError explains a 404 from 'traces show'. Job trace ids look exactly
// like endpoint trace ids in 'traces list --job' output, and the API has no
// detail endpoint for them, so the bare "not found" is misleading.
func traceShowError(err error) error {
	apiErr, ok := err.(*api.APIError)
	if !ok || apiErr.StatusCode != 404 {
		return err
	}
	return &api.APIError{
		StatusCode: apiErr.StatusCode,
		Message: apiErr.Message +
			" — note: background job traces have no detail endpoint; only endpoint (web request) traces can be shown in full",
	}
}
