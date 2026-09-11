package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var validMetricTypes = []string{"apdex", "response_time", "response_time_95th", "errors", "throughput", "queue_time"}

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Application metrics",
}

var metricsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get metric data with ASCII chart",
	Example: `  scout metrics get --type response_time --app 6
  scout metrics get --type throughput --app 6 --from 7d`,
	Run: runMetricsGet,
}

var metricTypeFlag string

func init() {
	metricsGetCmd.Flags().StringVar(&metricTypeFlag, "type", "", "Metric type ("+strings.Join(validMetricTypes, ", ")+")")
	_ = metricsGetCmd.MarkFlagRequired("type")
	metricsCmd.AddCommand(metricsGetCmd)
	rootCmd.AddCommand(metricsCmd)
}

func runMetricsGet(cmd *cobra.Command, args []string) {
	requireValidMetricType(metricTypeFlag)

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

	metrics, err := client.GetMetrics(id, metricTypeFlag, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(metrics) {
		return
	}

	series := metrics.Series[metricTypeFlag]
	summary := metrics.Summaries[metricTypeFlag]

	unit := unitForMetricType(metricTypeFlag)
	title := fmt.Sprintf("%s — %s", metricTypeFlag, appDisplayName(client, id))

	fmt.Println(output.RenderChart(title, series, summary, unit))
}

func isValidMetricType(t string) bool {
	return slices.Contains(validMetricTypes, t)
}

// requireValidMetricType exits with the valid list rather than round-tripping
// to the server for a 422.
func requireValidMetricType(t string) {
	if !isValidMetricType(t) {
		exitError(fmt.Sprintf("invalid metric type %q — valid types: %s",
			t, strings.Join(validMetricTypes, ", ")))
	}
}

// appDisplayName returns the app's name for a chart title, falling back to
// "App #<id>" when the name can't be fetched.
func appDisplayName(client *api.Client, id int) string {
	if app, err := client.GetApp(id); err == nil && app.Name != "" {
		return app.Name
	}
	return fmt.Sprintf("App #%d", id)
}

func unitForMetricType(t string) string {
	switch t {
	case "response_time", "response_time_95th", "queue_time":
		return "ms"
	case "throughput":
		return " rpm"
	case "errors":
		return ""
	case "apdex":
		return ""
	default:
		return ""
	}
}
