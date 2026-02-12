package cmd

import (
	"fmt"

	"github.com/scoutapm/scout-cli/internal/output"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Application metrics",
}

var metricsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get metric data with ASCII chart",
	Run:   runMetricsGet,
}

var metricTypeFlag string

func init() {
	metricsGetCmd.Flags().StringVar(&metricTypeFlag, "type", "", "Metric type (apdex, response_time, response_time_95th, errors, throughput, queue_time)")
	metricsGetCmd.MarkFlagRequired("type")
	metricsCmd.AddCommand(metricsGetCmd)
	rootCmd.AddCommand(metricsCmd)
}

func runMetricsGet(cmd *cobra.Command, args []string) {
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

	if jsonOutput {
		outputJSON(metrics)
		return
	}

	series := metrics.Series[metricTypeFlag]
	summary := metrics.Summaries[metricTypeFlag]

	unit := unitForMetricType(metricTypeFlag)
	title := fmt.Sprintf("%s — App #%d", metricTypeFlag, id)

	fmt.Println(output.RenderChart(title, series, summary, unit))
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
