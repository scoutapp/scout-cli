package cmd

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var validJobMetricTypes = []string{"throughput", "execution_time", "latency", "errors", "allocations"}

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Background job performance data",
	Long: `Background job performance data.

Job IDs are Base64 URL-safe encodings of a job's "queue/JobName" full name.
Get them from 'scout jobs list --json', or pass the full name
directly (e.g. --job default/MyWorker) and the CLI encodes it for you.

Use 'scout traces list --job <job-id>' to list traces for a background job.`,
}

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List background job performance",
	Run:   runJobsList,
}

var jobsMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Get job-specific metrics",
	Long: `Get time-series metrics for a single background job with an ASCII chart.

Valid metric types: ` + strings.Join(validJobMetricTypes, ", ") + `

For execution_time, the API returns one series per category (ActiveRecord,
Ruby, ...); the chart shows their per-timestamp total while --json
output keeps the per-category breakdown.`,
	Example: `  # By the Base64 job ID in the "job_id" field of 'scout jobs list --json'
  scout jobs metrics --job ZGVmYXVsdC9NeVdvcmtlcg== --type execution_time --app 6

  # The same job by its "queue/JobName" full name
  scout jobs metrics --job default/MyWorker --type throughput --app 6 --from 7d`,
	Run: runJobsMetrics,
}

var (
	jobFlag           string
	jobMetricTypeFlag string
)

func init() {
	jobsMetricsCmd.Flags().StringVar(&jobFlag, "job", "", "Job ID or queue/JobName full name")
	jobsMetricsCmd.Flags().StringVar(&jobMetricTypeFlag, "type", "", "Metric type ("+strings.Join(validJobMetricTypes, ", ")+")")
	_ = jobsMetricsCmd.MarkFlagRequired("job")
	_ = jobsMetricsCmd.MarkFlagRequired("type")
	jobsCmd.AddCommand(jobsListCmd, jobsMetricsCmd)
	rootCmd.AddCommand(jobsCmd)
}

func runJobsList(cmd *cobra.Command, args []string) {
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

	jobs, err := client.ListJobs(id, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	jobs, total := limitSlice(jobs)

	if structuredOutput(jobs) {
		return
	}

	headers := []string{"Job", "Queue", "Throughput", "Exec Time", "Latency", "% Time"}
	rows := make([][]string, len(jobs))
	for i := range jobs {
		j := jobs[i]
		rows[i] = []string{
			j.Name,
			j.Queue,
			output.FormatPerMinute(j.Throughput),
			output.FormatMs(j.ExecutionTime),
			output.FormatSeconds(j.Latency),
			formatTimeConsumed(j.TimeConsumed),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(len(jobs), total)
}

func runJobsMetrics(cmd *cobra.Command, args []string) {
	if !isValidJobMetricType(jobMetricTypeFlag) {
		exitError(fmt.Sprintf("invalid job metric type %q — valid types: %s",
			jobMetricTypeFlag, strings.Join(validJobMetricTypes, ", ")))
	}

	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	jobFlag, err := requireFlagValue("job", jobFlag)
	if err != nil {
		exitError(err.Error())
	}

	jobID, err := resolveJobID(jobFlag)
	if err != nil {
		exitError(err.Error())
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	metrics, err := client.GetJobMetrics(id, jobID, jobMetricTypeFlag, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(metrics) {
		return
	}

	unit := unitForJobMetricType(jobMetricTypeFlag)
	title := fmt.Sprintf("%s — %s", jobMetricTypeFlag, jobDisplayName(jobFlag))

	fmt.Println(output.RenderChart(title, chartSeries(jobMetricTypeFlag, metrics), metrics.Summary, unit))
}

// chartSeries picks the series to chart for a job metric. The latency
// response carries a "total" sub-series that is the job's total execution
// time rather than queue latency, so chart the "Latency" series itself; every
// other type uses the normalized total.
func chartSeries(metricType string, m *api.JobMetricsResult) []api.MetricPoint {
	if metricType == "latency" {
		if pts, ok := m.Series["Latency"]; ok {
			return pts
		}
	}
	return m.Total()
}

// resolveJobID accepts either an already-encoded job ID or a "queue/JobName"
// full name and returns the Base64 URL-safe job ID the API expects (matching
// Ruby's Base64.urlsafe_encode64, including padding).
//
// A value that is neither form — most often a job class name copy-pasted
// without its queue — is rejected here rather than passed through to the
// server, which answers an unrecognized id with a bare 500.
func resolveJobID(s string) (string, error) {
	if strings.Contains(s, "/") {
		return base64.URLEncoding.EncodeToString([]byte(s)), nil
	}
	if name, ok := decodeBase64ID(s); !ok || !strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid job %q — expected a job id from 'scout jobs list --json' (the job_id field), or a full name like default/MyWorker", s)
	}
	return s, nil
}

// jobDisplayName returns a human-friendly job name for titles: the full name
// as given, the decoded full name for an encoded ID, or the raw input.
func jobDisplayName(s string) string {
	if strings.Contains(s, "/") {
		return s
	}
	if name, ok := decodeBase64ID(s); ok {
		return name
	}
	return s
}

func isValidJobMetricType(t string) bool {
	for _, v := range validJobMetricTypes {
		if v == t {
			return true
		}
	}
	return false
}

func unitForJobMetricType(t string) string {
	switch t {
	case "throughput":
		return "/min"
	case "execution_time", "latency":
		return "ms"
	default:
		return ""
	}
}

// formatTimeConsumed renders a 0–1 fraction as a percentage.
func formatTimeConsumed(fraction float64) string {
	return fmt.Sprintf("%.1f%%", fraction*100)
}
