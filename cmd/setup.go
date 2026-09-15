package cmd

import (
	"fmt"

	"github.com/scoutapm/scout/internal/output"
	"github.com/spf13/cobra"
)

// Scout's docs are organized by language, not by framework, and several
// frameworks have no dedicated page — so each entry carries its own URL
// rather than deriving one from the framework name.
var frameworks = []struct {
	name    string
	desc    string
	docsURL string
}{
	{"rails", "Ruby on Rails", "https://scoutapm.com/docs/ruby"},
	{"django", "Python Django", "https://scoutapm.com/docs/python/django"},
	{"flask", "Python Flask", "https://scoutapm.com/docs/python/flask"},
	{"phoenix", "Elixir Phoenix", "https://scoutapm.com/docs/elixir"},
	{"express", "Node.js Express", "https://scoutapm.com/docs/node/express"},
	{"laravel", "PHP Laravel", "https://scoutapm.com/docs/php/laravel"},
	{"sinatra", "Ruby Sinatra", "https://scoutapm.com/docs/ruby/sinatra"},
	{"fastapi", "Python FastAPI", "https://scoutapm.com/docs/python/fastapi"},
	{"celery", "Python Celery", "https://scoutapm.com/docs/python/celery"},
	{"dramatiq", "Python Dramatiq", "https://scoutapm.com/docs/python/other-libraries#dramatiq"},
	{"sidekiq", "Ruby Sidekiq", "https://scoutapm.com/docs/ruby#instrumented-libraries"},
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

func frameworkDocsURL(name string) (string, bool) {
	for _, f := range frameworks {
		if f.name == name {
			return f.docsURL, true
		}
	}
	return "", false
}

func runSetup(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		names := make([]string, len(frameworks))
		for i, f := range frameworks {
			names[i] = f.name
		}
		if structuredOutput(map[string]interface{}{"frameworks": names}) {
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
	docsURL, found := frameworkDocsURL(framework)
	if !found {
		exitError(fmt.Sprintf("unknown framework: %s", framework))
	}

	if structuredOutput(map[string]interface{}{
		"framework": framework,
		"docs_url":  docsURL,
	}) {
		return
	}

	fmt.Println(output.HeaderStyle.Render(fmt.Sprintf("Setup: %s", framework)))
	fmt.Println()
	fmt.Printf("  Documentation: %s\n", docsURL)
	fmt.Println()
	fmt.Println(output.DimStyle.Render("  Visit the link above for detailed installation instructions."))
}
