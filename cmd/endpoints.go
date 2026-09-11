package cmd

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/scoutapm/scout/internal/output"
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
	Example: `  # The endpoint ID is the Base64 value at the end of the "link"
  # field in 'scout endpoints list --json'
  scout endpoints metrics --endpoint YXBpL21ldHJpY3Mvc2hvdw== --type response_time --app 6`,
	Run: runEndpointsMetrics,
}

var endpointFlag string
var endpointMetricTypeFlag string

func init() {
	endpointsMetricsCmd.Flags().StringVar(&endpointFlag, "endpoint", "", "URL-encoded endpoint name")
	endpointsMetricsCmd.Flags().StringVar(&endpointMetricTypeFlag, "type", "", "Metric type ("+strings.Join(validMetricTypes, ", ")+")")
	_ = endpointsMetricsCmd.MarkFlagRequired("endpoint")
	_ = endpointsMetricsCmd.MarkFlagRequired("type")
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

	if structuredOutput(endpoints) {
		return
	}

	total := len(endpoints)
	limit, _ := applyLimit(total)

	headers := []string{"Name", "Resp Time", "Throughput", "Error %", "p95"}
	rows := make([][]string, limit)
	for i := 0; i < limit; i++ {
		ep := endpoints[i]
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
	printTruncated(limit, total)
}

func runEndpointsMetrics(cmd *cobra.Command, args []string) {
	requireValidMetricType(endpointMetricTypeFlag)

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

	if structuredOutput(metrics) {
		return
	}

	series := metrics.Series[endpointMetricTypeFlag]
	summary := metrics.Summaries[endpointMetricTypeFlag]
	unit := unitForMetricType(endpointMetricTypeFlag)
	title := fmt.Sprintf("%s — %s", endpointMetricTypeFlag, endpointDisplayName(endpointFlag))

	fmt.Println(output.RenderChart(title, series, summary, unit))
}

// endpointDisplayName decodes a Base64 URL-safe endpoint ID back to the
// endpoint name for display, returning the input unchanged when it isn't one.
func endpointDisplayName(id string) string {
	b, err := base64.URLEncoding.DecodeString(id)
	if err != nil {
		b, err = base64.RawURLEncoding.DecodeString(id)
		if err != nil {
			return id
		}
	}
	name := string(b)
	if name == "" || !utf8.ValidString(name) {
		return id
	}
	for _, r := range name {
		if !unicode.IsPrint(r) {
			return id
		}
	}
	return name
}
