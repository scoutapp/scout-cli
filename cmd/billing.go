package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Show usage for the current billing period",
	Long: `Show the organization's usage for the current billing period, as reported
by the Scout API's /usage endpoint.

Includes the billing period dates, pricing style, APM transaction count (with
the plan limit when one applies), active node count (per-node pricing), error
count (when the errors add-on is enabled), and log bytes ingested (when a logs
integration is enabled).

Unlike 'scout usage', which estimates web transactions from throughput
metrics, these figures are the exact values Scout bills against and include
background jobs.`,
	Run: runBilling,
}

func init() {
	rootCmd.AddCommand(billingCmd)
}

func runBilling(cmd *cobra.Command, args []string) {
	client, err := getClient()
	if err != nil {
		exitError(err.Error())
	}

	usage, err := client.GetOrgUsage()
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(usage) {
		return
	}

	fmt.Print(renderBillingSummary(usage))
}

// renderBillingSummary formats the org usage payload as a human-readable
// report with one block per section the API returned.
func renderBillingSummary(u *api.OrgUsage) string {
	var sections []string

	var period []string
	if u.BillingPeriod.Start == "" || u.BillingPeriod.End == "" {
		period = append(period, output.DimStyle.Render("not available"))
	} else {
		line := fmt.Sprintf("%s → %s", formatBillingDate(u.BillingPeriod.Start), formatBillingDate(u.BillingPeriod.End))
		if remaining := daysRemaining(u.BillingPeriod.End); remaining >= 0 {
			line += "  " + output.DimStyle.Render(fmt.Sprintf("(%d days remaining)", remaining))
		}
		period = append(period, line)
	}
	if u.PricingStyle != "" {
		period = append(period, "Pricing: "+u.PricingStyle)
	}
	sections = append(sections, renderSection("Billing Period", period))

	if u.APM != nil {
		lines := []string{"Total: " + formatTransactions(float64(u.APM.TotalTransactions))}
		if u.APM.Limit != nil && *u.APM.Limit > 0 {
			lines = append(lines,
				"Limit: "+formatTransactions(float64(*u.APM.Limit)),
				renderUsageBar(u.APM.TotalTransactions, *u.APM.Limit),
			)
		}
		sections = append(sections, renderSection("APM Transactions", lines))
	}

	if u.Nodes != nil {
		sections = append(sections, renderSection("Nodes", []string{
			fmt.Sprintf("Active: %d", u.Nodes.ActiveCount),
		}))
	}

	if u.Errors != nil {
		lines := []string{"Count: " + formatTransactions(float64(u.Errors.Count))}
		if u.Errors.Limit > 0 {
			lines = append(lines,
				"Limit: "+formatTransactions(float64(u.Errors.Limit)),
				renderUsageBar(u.Errors.Count, u.Errors.Limit),
			)
		}
		sections = append(sections, renderSection("Errors", lines))
	}

	if u.Logs != nil {
		lines := []string{"Used: " + output.FormatBytes(u.Logs.BytesUsed)}
		if u.Logs.LimitBytes != nil && *u.Logs.LimitBytes > 0 {
			lines = append(lines,
				"Limit: "+output.FormatBytes(*u.Logs.LimitBytes),
				renderUsageBar(u.Logs.BytesUsed, *u.Logs.LimitBytes),
			)
		}
		sections = append(sections, renderSection("Logs", lines))
	}

	return strings.Join(sections, "\n") + "\n"
}

// renderSection renders a bold title followed by indented lines.
func renderSection(title string, lines []string) string {
	var sb strings.Builder
	sb.WriteString(output.BoldStyle.Render(title))
	sb.WriteString("\n")
	for _, l := range lines {
		sb.WriteString("  ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderUsageBar renders an ASCII progress bar like [████████░░░░░░░░░░░░] 42.0%
func renderUsageBar(used, limit int64) string {
	const barWidth = 20
	pct := float64(used) / float64(limit) * 100
	filled := int(math.Round(float64(barWidth) * float64(used) / float64(limit)))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	// Anything under 2.5% rounds to no blocks at all, which reads as zero
	// usage. Show one block instead so "a little" is distinguishable.
	if filled == 0 && used > 0 {
		filled = 1
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	if pct > 100 {
		return output.WarningStyle.Render(fmt.Sprintf("[%s] %.1f%% (over limit)", bar, pct))
	}
	return fmt.Sprintf("[%s] %.1f%%", bar, pct)
}

// billingDateLayouts are the formats the API may use for billing period dates.
var billingDateLayouts = []string{time.RFC3339, "2006-01-02"}

// parseBillingTime parses an ISO 8601 date or datetime.
func parseBillingTime(s string) (time.Time, error) {
	for _, layout := range billingDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as date", s)
}

// formatBillingDate converts an ISO 8601 date string to "Mar 01, 2026" format.
// Unparseable input is returned unchanged.
func formatBillingDate(iso string) string {
	t, err := parseBillingTime(iso)
	if err != nil {
		return iso
	}
	return t.Format("Jan 02, 2006")
}

// daysRemaining calculates whole days until the given ISO 8601 date. Returns 0
// for dates in the past and -1 if the input is unparseable.
func daysRemaining(endISO string) int {
	t, err := parseBillingTime(endISO)
	if err != nil {
		return -1
	}
	days := int(time.Until(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
