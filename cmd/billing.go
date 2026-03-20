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
	Short: "Show billing period usage summary",
	Run:   runBilling,
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

	if jsonOutput {
		outputJSON(usage)
		return
	}

	printBillingSummary(usage)
}

func printBillingSummary(u *api.OrgUsage) {
	// Billing period header
	startStr := formatBillingDate(u.BillingPeriod.Start)
	endStr := formatBillingDate(u.BillingPeriod.End)
	remaining := daysRemaining(u.BillingPeriod.End)

	fmt.Println(output.BoldStyle.Render("Billing Period"))
	fmt.Printf("  %s → %s", startStr, endStr)
	if remaining >= 0 {
		fmt.Printf("  %s", output.DimStyle.Render(fmt.Sprintf("(%d days remaining)", remaining)))
	}
	fmt.Println()
	fmt.Printf("  Pricing: %s\n", u.PricingStyle)
	fmt.Println()

	// APM
	if u.APM != nil {
		fmt.Println(output.BoldStyle.Render("APM Transactions"))
		fmt.Printf("  Total: %s\n", formatTransactions(float64(u.APM.TotalTransactions)))
		if u.APM.Limit != nil && *u.APM.Limit > 0 {
			fmt.Printf("  Limit: %s\n", formatTransactions(float64(*u.APM.Limit)))
			fmt.Printf("  %s\n", renderUsageBar(u.APM.TotalTransactions, *u.APM.Limit))
		}
		fmt.Println()
	}

	// Nodes
	if u.Nodes != nil {
		fmt.Println(output.BoldStyle.Render("Nodes"))
		fmt.Printf("  Active: %d\n", u.Nodes.ActiveCount)
		fmt.Println()
	}

	// Errors
	if u.Errors != nil {
		fmt.Println(output.BoldStyle.Render("Errors"))
		fmt.Printf("  Count: %s\n", formatTransactions(float64(u.Errors.Count)))
		if u.Errors.Limit > 0 {
			fmt.Printf("  Limit: %s\n", formatTransactions(float64(u.Errors.Limit)))
			fmt.Printf("  %s\n", renderUsageBar(u.Errors.Count, u.Errors.Limit))
		}
		fmt.Println()
	}

	// Logs
	if u.Logs != nil {
		fmt.Println(output.BoldStyle.Render("Logs"))
		fmt.Printf("  Used: %s\n", output.FormatBytes(u.Logs.BytesUsed))
		if u.Logs.LimitBytes != nil && *u.Logs.LimitBytes > 0 {
			fmt.Printf("  Limit: %s\n", output.FormatBytes(*u.Logs.LimitBytes))
			fmt.Printf("  %s\n", renderUsageBar(u.Logs.BytesUsed, *u.Logs.LimitBytes))
		}
		fmt.Println()
	}
}

// renderUsageBar renders an ASCII progress bar like [████████░░░░░░░░░░░░] 42%
func renderUsageBar(used, limit int64) string {
	const barWidth = 20
	pct := float64(used) / float64(limit) * 100
	filled := int(math.Round(float64(barWidth) * float64(used) / float64(limit)))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	if pct > 100 {
		return output.WarningStyle.Render(fmt.Sprintf("[%s] %.1f%% (over limit)", bar, pct))
	}
	return fmt.Sprintf("[%s] %.1f%%", bar, pct)
}

// formatBillingDate converts an ISO 8601 date string to "Mar 01, 2026" format.
func formatBillingDate(iso string) string {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, iso); err == nil {
			return t.Format("Jan 02, 2006")
		}
	}
	return iso
}

// daysRemaining calculates days until the given ISO 8601 date. Returns -1 if unparseable.
func daysRemaining(endISO string) int {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, endISO); err == nil {
			days := int(time.Until(t).Hours() / 24)
			if days < 0 {
				return 0
			}
			return days
		}
	}
	return -1
}
