package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/scoutapm/scout/internal/api"
	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Performance insights (N+1, memory bloat, slow queries)",
}

var insightsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all insights grouped by type",
	Run:   runInsightsList,
}

var insightsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show insights of a specific type",
	Run:   runInsightsShow,
}

var insightTypeFlag string

func init() {
	insightsShowCmd.Flags().StringVar(&insightTypeFlag, "type", "", "Insight type (n_plus_one, memory_bloat, slow_query)")
	_ = insightsShowCmd.MarkFlagRequired("type")
	insightsCmd.AddCommand(insightsListCmd, insightsShowCmd)
	rootCmd.AddCommand(insightsCmd)
}

func runInsightsList(cmd *cobra.Command, args []string) {
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

	result, err := client.ListInsights(id, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	if structuredOutput(limitInsightCategories(result)) {
		return
	}

	types := []struct {
		key   string
		label string
	}{
		{"n_plus_one", "N+1 Queries"},
		{"memory_bloat", "Memory Bloat"},
		{"slow_query", "Slow Queries"},
	}

	for _, t := range types {
		cat, ok := result.Insights[t.key]
		if !ok {
			continue
		}

		header := fmt.Sprintf("%s (%d total, %d new)", t.label, cat.Count, cat.NewCount)
		if cat.Count == 0 {
			fmt.Printf("%s %s\n", output.SuccessStyle.Render("✓"), output.DimStyle.Render(header))
		} else {
			fmt.Println(output.WarningStyle.Render(header))
			items, total := limitSlice(cat.Items)
			for _, item := range items {
				fmt.Printf("  %s %s\n", output.WarningStyle.Render("⚠"), item.Name)
			}
			printTruncated(len(items), total)
		}
		fmt.Println()
	}
}

// limitInsightCategories applies -n to each category's items. The category
// counts describe what exists and are left alone; -n only selects how many
// items are listed.
func limitInsightCategories(result *api.InsightsListResult) *api.InsightsListResult {
	limited := *result
	limited.Insights = make(map[string]api.InsightCategory, len(result.Insights))
	for key, cat := range result.Insights {
		cat.Items, _ = limitSlice(cat.Items)
		limited.Insights[key] = cat
	}
	return &limited
}

func runInsightsShow(cmd *cobra.Command, args []string) {
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

	result, err := client.GetInsightsByType(id, insightTypeFlag, from, to)
	if err != nil {
		handleAPIError(err)
		return
	}

	items, total := limitSlice(result.Items)
	limited := *result
	limited.Items = items

	if structuredOutput(&limited) {
		return
	}

	header := fmt.Sprintf("%s — %d total, %d new", result.InsightType, result.TotalCount, result.NewCount)
	fmt.Println(output.HeaderStyle.Render(header))
	fmt.Println()

	for _, item := range items {
		fmt.Println(output.BoldStyle.Render(item.Name))
		printInsightFields(item.Fields, "  ")
		fmt.Println()
	}
	printTruncated(len(items), total)
}

// printInsightFields prints an insight's fields in key order. The API returns
// them as a map, so without sorting the same insight prints in a different
// order every run; nested objects are indented rather than left to Go's
// map[...] formatting.
func printInsightFields(fields map[string]interface{}, indent string) {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		label := output.DimStyle.Render(k + ":")
		nested, ok := fields[k].(map[string]interface{})
		if ok && len(nested) > 0 {
			fmt.Printf("%s%s\n", indent, label)
			printInsightFields(nested, indent+"  ")
			continue
		}
		fmt.Printf("%s%-30s %s\n", indent, label, formatInsightValue(fields[k]))
	}
}

// formatInsightValue renders a leaf field value for display, standing in for
// values Go would print as <nil> or as raw syntax.
func formatInsightValue(v interface{}) string {
	switch tv := v.(type) {
	case nil:
		return "—"
	case string:
		if tv == "" {
			return "—"
		}
		return tv
	case []interface{}:
		if len(tv) == 0 {
			return "—"
		}
		parts := make([]string, len(tv))
		for i, e := range tv {
			parts[i] = formatInsightValue(e)
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		return "—"
	default:
		return fmt.Sprintf("%v", v)
	}
}
