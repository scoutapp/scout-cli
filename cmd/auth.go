package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/scoutapm/scout-cli/internal/api"
	"github.com/scoutapm/scout-cli/internal/config"
	"github.com/scoutapm/scout-cli/internal/output"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Scout APM",
	Run:   runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	Run:   runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Run:   runAuthStatus,
}

var loginKeyFlag string

func init() {
	authLoginCmd.Flags().StringVar(&loginKeyFlag, "key", "", "API key (interactive prompt if omitted)")
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

// BubbleTea model for interactive login
type loginModel struct {
	textInput textinput.Model
	spinner   spinner.Model
	apiKey    string
	state     string // "input", "validating", "done", "error"
	appCount  int
	err       error
}

type validateResult struct {
	count int
	err   error
}

func initialLoginModel() loginModel {
	ti := textinput.New()
	ti.Placeholder = "Enter your Scout API key"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Focus()

	s := spinner.New()
	s.Spinner = spinner.Dot

	return loginModel{
		textInput: ti,
		spinner:   s,
		state:     "input",
	}
}

func (m loginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.state == "input" {
				m.apiKey = m.textInput.Value()
				if m.apiKey == "" {
					return m, nil
				}
				m.state = "validating"
				return m, tea.Batch(m.spinner.Tick, validateKey(m.apiKey))
			}
		}
	case spinner.TickMsg:
		if m.state == "validating" {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case validateResult:
		if msg.err != nil {
			m.state = "error"
			m.err = msg.err
		} else {
			m.state = "done"
			m.appCount = msg.count
		}
		return m, tea.Quit
	}

	if m.state == "input" {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m loginModel) View() string {
	switch m.state {
	case "input":
		return fmt.Sprintf("? %s\n", m.textInput.View())
	case "validating":
		return fmt.Sprintf("  Validating... %s\n", m.spinner.View())
	case "done":
		return output.SuccessStyle.Render(fmt.Sprintf("✓ Authenticated — %d apps found", m.appCount)) + "\n"
	case "error":
		return output.ErrorStyle.Render(fmt.Sprintf("✗ %s", m.err.Error())) + "\n"
	}
	return ""
}

func validateKey(key string) tea.Cmd {
	return func() tea.Msg {
		client := api.NewClient(config.GetAPIURL(), key)
		apps, err := client.ListApps()
		if err != nil {
			return validateResult{err: err}
		}
		return validateResult{count: len(apps)}
	}
}

func runAuthLogin(cmd *cobra.Command, args []string) {
	key := loginKeyFlag
	if key == "" {
		// Interactive mode with BubbleTea
		p := tea.NewProgram(initialLoginModel())
		finalModel, err := p.Run()
		if err != nil {
			exitError(err.Error())
		}
		m := finalModel.(loginModel)
		if m.state == "error" {
			os.Exit(2)
		}
		if m.state != "done" {
			os.Exit(1)
		}
		key = m.apiKey
	} else {
		// Non-interactive: validate and save
		client := api.NewClient(config.GetAPIURL(), key)
		apps, err := client.ListApps()
		if err != nil {
			handleAPIError(err)
			return
		}
		fmt.Println(output.SuccessStyle.Render(fmt.Sprintf("✓ Authenticated — %d apps found", len(apps))))
	}

	cfg, _ := config.Read()
	cfg.APIKey = key
	if err := config.Write(cfg); err != nil {
		exitError(fmt.Sprintf("failed to save config: %s", err))
	}
}

func runAuthLogout(cmd *cobra.Command, args []string) {
	if err := config.Clear(); err != nil {
		exitError(fmt.Sprintf("failed to clear config: %s", err))
	}
	fmt.Printf("Credentials cleared from %s\n", config.Path())
}

func runAuthStatus(cmd *cobra.Command, args []string) {
	key := config.GetAPIKey()
	url := config.GetAPIURL()
	cfg, _ := config.Read()

	if jsonOutput {
		data := map[string]interface{}{
			"authenticated":  key != "",
			"api_key_prefix": maskKey(key),
			"api_url":        url,
		}
		if cfg.DefaultAppID > 0 {
			data["default_app_id"] = cfg.DefaultAppID
		}
		outputJSON(data)
		return
	}

	if key == "" {
		fmt.Println(output.WarningStyle.Render("Not authenticated"))
		fmt.Println(output.DimStyle.Render("Run 'scout auth login' to authenticate"))
		return
	}

	fmt.Println(output.SuccessStyle.Render("✓ Authenticated"))
	fmt.Printf("  API Key: %s\n", maskKey(key))
	fmt.Printf("  API URL: %s\n", url)
	if cfg.DefaultAppID > 0 {
		fmt.Printf("  Default App: %d\n", cfg.DefaultAppID)
	}
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}
