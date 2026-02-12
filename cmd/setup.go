package cmd

import (
	"fmt"

	"github.com/scoutapm/scout-cli/internal/output"
	"github.com/spf13/cobra"
)

var frameworks = []struct {
	name string
	desc string
}{
	{"rails", "Ruby on Rails"},
	{"django", "Python Django"},
	{"flask", "Python Flask"},
	{"phoenix", "Elixir Phoenix"},
	{"express", "Node.js Express"},
	{"laravel", "PHP Laravel"},
	{"sinatra", "Ruby Sinatra"},
	{"fastapi", "Python FastAPI"},
	{"celery", "Python Celery"},
	{"dramatiq", "Python Dramatiq"},
	{"sidekiq", "Ruby Sidekiq"},
}

var setupCmd = &cobra.Command{
	Use:   "setup [framework]",
	Short: "Show setup instructions for a framework",
	Args:  cobra.MaximumNArgs(1),
	Run:   runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		if jsonOutput {
			names := make([]string, len(frameworks))
			for i, f := range frameworks {
				names[i] = f.name
			}
			outputJSON(map[string]interface{}{"frameworks": names})
			return
		}

		fmt.Println(output.HeaderStyle.Render("Supported Frameworks"))
		fmt.Println()
		for _, f := range frameworks {
			fmt.Printf("  %s  %s\n",
				output.BoldStyle.Render(fmt.Sprintf("%-12s", f.name)),
				output.DimStyle.Render(f.desc),
			)
		}
		fmt.Println()
		fmt.Println(output.DimStyle.Render("Usage: scout setup <framework>"))
		return
	}

	framework := args[0]
	found := false
	for _, f := range frameworks {
		if f.name == framework {
			found = true
			break
		}
	}

	if !found {
		exitError(fmt.Sprintf("unknown framework: %s", framework))
	}

	docsURL := fmt.Sprintf("https://scoutapm.com/docs/%s", framework)

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"framework": framework,
			"docs_url":  docsURL,
		})
		return
	}

	fmt.Println(output.HeaderStyle.Render(fmt.Sprintf("Setup: %s", framework)))
	fmt.Println()
	fmt.Printf("  Documentation: %s\n", docsURL)
	fmt.Println()
	fmt.Println(output.DimStyle.Render("  Visit the link above for detailed installation instructions."))
}
