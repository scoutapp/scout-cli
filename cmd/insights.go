package cmd

import (
	"fmt"

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
	insightsShowCmd.MarkFlagRequired("type")
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

	if jsonOutput {
		outputJSON(result)
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
			for _, item := range cat.Items {
				fmt.Printf("  %s %s\n", output.WarningStyle.Render("⚠"), item.Name)
			}
		}
		fmt.Println()
	}
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

	if jsonOutput {
		outputJSON(result)
		return
	}

	header := fmt.Sprintf("%s — %d total, %d new", result.InsightType, result.TotalCount, result.NewCount)
	fmt.Println(output.HeaderStyle.Render(header))
	fmt.Println()

	for _, item := range result.Items {
		fmt.Println(output.BoldStyle.Render(item.Name))
		for k, v := range item.Fields {
			fmt.Printf("  %-30s %v\n", output.DimStyle.Render(k+":"), v)
		}
		fmt.Println()
	}
}
