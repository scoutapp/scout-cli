package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var (
	anomaliesState    string
	anomaliesMetric   string
	anomaliesEndpoint string
)

var anomaliesCmd = &cobra.Command{
	Use:   "anomalies",
	Short: "List and inspect anomaly events",
	Run:   runAnomaliesList,
}

var anomaliesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List anomaly events",
	Run:   runAnomaliesList,
}

var anomaliesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show anomaly event details",
	Args:  cobra.ExactArgs(1),
	Run:   runAnomaliesShow,
}

func init() {
	for _, c := range []*cobra.Command{anomaliesCmd, anomaliesListCmd} {
		c.Flags().StringVar(&anomaliesState, "state", "all", "Filter by state: open|closed|all")
		c.Flags().StringVar(&anomaliesMetric, "metric", "", "Filter by metric (e.g. response_time)")
		c.Flags().StringVar(&anomaliesEndpoint, "endpoint", "", "Filter by endpoint")
	}

	anomaliesCmd.AddCommand(anomaliesListCmd, anomaliesShowCmd)
	rootCmd.AddCommand(anomaliesCmd)
}

func runAnomaliesList(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	endpoint := ""
	if cmd.Flags().Changed("endpoint") {
		endpoint, err = resolveAnomalyEndpoint(anomaliesEndpoint)
		if err != nil {
			exitError(err.Error())
		}
	}

	from, to, err := resolveTimeframe()
	if err != nil {
		exitError(err.Error())
	}

	events, err := client.ListAnomalyEvents(id, anomaliesState, anomaliesMetric, endpoint, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	events, total := limitSlice(events)

	if structuredOutput(events) {
		return
	}

	headers := []string{"ID", "State", "Metric", "Endpoint", "Started", "Z-score", "Multiplier"}
	rows := make([][]string, len(events))
	for i := range events {
		e := events[i]
		state := anomalyState(e.Open)
		rows[i] = []string{
			strconv.Itoa(e.ID),
			output.StatusColor(state).Render(state),
			e.Metric,
			e.Endpoint,
			output.FormatRelativeTime(e.StartedAt),
			fmt.Sprintf("%.1f", e.ZScore),
			formatMultiplier(e.Multiplier),
		}
	}

	fmt.Println(output.RenderTable(headers, rows))
	printTruncated(len(events), total)
}

// resolveAnomalyEndpoint accepts the endpoint forms used elsewhere in the CLI
// and returns the scoped metric name the anomaly events API matches on.
//
// Anomalies are recorded against the full metric name ("Controller/users/index")
// shown in 'scout anomalies list', while every other --endpoint flag takes the
// Base64 URL-safe endpoint id from 'scout endpoints list'. Both are accepted
// here, as is the plain endpoint name, so a base64 id no longer silently
// matches nothing.
func resolveAnomalyEndpoint(s string) (string, error) {
	name, err := requireFlagValue("endpoint", s)
	if err != nil {
		return "", err
	}
	if !strings.Contains(name, "/") {
		decoded, ok := decodeBase64ID(name)
		if !ok {
			return "", fmt.Errorf("invalid --endpoint %q — expected an endpoint name from 'scout anomalies list' (e.g. Controller/users/index) or a Base64 endpoint id from 'scout endpoints list --json'", s)
		}
		name = decoded
	}
	if hasMetricScope(name) {
		return name, nil
	}
	return "Controller/" + name, nil
}

// hasMetricScope reports whether a name already carries a metric scope prefix
// such as "Controller/" or "Job/". Endpoint names on their own are the
// lowercase path portion ("users/index").
func hasMetricScope(name string) bool {
	scope, rest, ok := strings.Cut(name, "/")
	if !ok || scope == "" || rest == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(scope)
	return unicode.IsUpper(r)
}

func runAnomaliesShow(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	id, err := requireAppID()
	if err != nil {
		exitError(err.Error())
	}

	eventID, err := strconv.Atoi(args[0])
	if err != nil {
		exitError("invalid anomaly event ID: " + args[0])
	}

	event, err := client.GetAnomalyEvent(id, eventID)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(event) {
		return
	}

	state := anomalyState(event.Open)
	fmt.Println(output.HeaderStyle.Render(fmt.Sprintf("Anomaly #%d", event.ID)))
	fmt.Printf("  State:       %s\n", output.StatusColor(state).Render(state))
	fmt.Printf("  Metric:      %s\n", event.Metric)
	if event.Endpoint != "" {
		fmt.Printf("  Endpoint:    %s\n", event.Endpoint)
	}
	fmt.Printf("  Direction:   %s\n", event.Direction)
	fmt.Printf("  Severity:    %s\n", event.Severity)
	fmt.Printf("  Started:     %s\n", output.FormatRelativeTime(event.StartedAt))
	if event.EndedAt != nil {
		fmt.Printf("  Ended:       %s\n", output.FormatRelativeTime(*event.EndedAt))
	}
	fmt.Printf("  Last seen:   %s\n", output.FormatRelativeTime(event.LastSeenAt))
	fmt.Printf("  Z-score:     %.2f\n", event.ZScore)
	fmt.Printf("  Current:     %.2f\n", event.CurrentValue)
	fmt.Printf("  Baseline:    %.2f\n", event.BaselineValue)
	fmt.Printf("  Multiplier:  %s\n", formatMultiplier(event.Multiplier))
	if event.BaselineStdDev != nil {
		fmt.Printf("  Std dev:     %.2f\n", *event.BaselineStdDev)
	}
	if event.DurationMinutes != nil {
		fmt.Printf("  Duration:    %d min\n", *event.DurationMinutes)
	}
	if event.Description != "" {
		fmt.Printf("  Description: %s\n", event.Description)
	}

	if event.SmartMonitor != nil {
		fmt.Println()
		fmt.Println(output.BoldStyle.Render("Smart Monitor"))
		fmt.Printf("  ID:                 %d\n", event.SmartMonitor.ID)
		fmt.Printf("  Name:               %s\n", event.SmartMonitor.Name)
		fmt.Printf("  Kind:               %s\n", event.SmartMonitor.Kind)
		fmt.Printf("  Severity threshold: %.1f\n", event.SmartMonitor.SeverityThreshold)
		fmt.Printf("  Duration:           %d min\n", event.SmartMonitor.DurationMinutes)
	}

	if event.Deploy != nil {
		fmt.Println()
		fmt.Println(output.BoldStyle.Render("Deploy"))
		fmt.Printf("  ID:       %d\n", event.Deploy.ID)
		fmt.Printf("  SHA:      %s\n", event.Deploy.SHA)
		fmt.Printf("  Deployed: %s\n", output.FormatRelativeTime(event.Deploy.DeployedAt))
	}
}

func formatMultiplier(m *float64) string {
	if m == nil {
		return "—"
	}
	return fmt.Sprintf("%.1fx", *m)
}

func anomalyState(open bool) string {
	if open {
		return "open"
	}
	return "closed"
}
