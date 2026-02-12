package cmd

import (
	"fmt"

	"github.com/scoutapm/scout-cli/internal/output"
	"github.com/spf13/cobra"
)

var endpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "Endpoint performance data",
}

var endpointsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List endpoint performance",
	Run:   runEndpointsList,
}

var endpointsMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Get endpoint-specific metrics",
	Run:   runEndpointsMetrics,
}

var endpointFlag string
var endpointMetricTypeFlag string

func init() {
	endpointsMetricsCmd.Flags().StringVar(&endpointFlag, "endpoint", "", "URL-encoded endpoint name")
	endpointsMetricsCmd.Flags().StringVar(&endpointMetricTypeFlag, "type", "", "Metric type")
	endpointsMetricsCmd.MarkFlagRequired("endpoint")
	endpointsMetricsCmd.MarkFlagRequired("type")
	endpointsCmd.AddCommand(endpointsListCmd, endpointsMetricsCmd)
	rootCmd.AddCommand(endpointsCmd)
}

func runEndpointsList(cmd *cobra.Command, args []string) {
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

	endpoints, err := client.ListEndpoints(id, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(endpoints)
		return
	}

	headers := []string{"Name", "Resp Time", "Throughput", "Error %", "p95"}
	rows := make([][]string, len(endpoints))
	for i, ep := range endpoints {
		errorPct := output.FormatPercent(ep.ErrorRate)
		errorColored := output.ErrorRateColor(ep.ErrorRate).Render(errorPct)

		rows[i] = []string{
			ep.FormattedMethodName,
			output.FormatMs(ep.ResponseTime),
			output.FormatRPM(ep.Throughput),
			errorColored,
			output.FormatMs(ep.P95),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
}

func runEndpointsMetrics(cmd *cobra.Command, args []string) {
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

	metrics, err := client.GetEndpointMetrics(id, endpointFlag, endpointMetricTypeFlag, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if jsonOutput {
		outputJSON(metrics)
		return
	}

	series := metrics.Series[endpointMetricTypeFlag]
	summary := metrics.Summaries[endpointMetricTypeFlag]
	unit := unitForMetricType(endpointMetricTypeFlag)
	title := fmt.Sprintf("%s — %s", endpointMetricTypeFlag, endpointFlag)

	fmt.Println(output.RenderChart(title, series, summary, unit))
}
