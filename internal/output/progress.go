package output

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type progressMsg float64

type doneMsg[T any] struct {
	result T
}

type progressModel[T any] struct {
	label    string
	bar      progress.Model
	result   T
	done     bool
}

func (m progressModel[T]) Init() tea.Cmd {
	return nil
}

func (m progressModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case progressMsg:
		return m, m.bar.SetPercent(float64(msg))
	case doneMsg[T]:
		m.result = msg.result
		m.done = true
		return m, tea.Quit
	case progress.FrameMsg:
		barModel, cmd := m.bar.Update(msg)
		m.bar = barModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m progressModel[T]) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s  %s", DimStyle.Render(m.label), m.bar.View())
}

// RunWithProgress runs fn while displaying an animated progress bar.
// fn receives an update callback that accepts a value between 0.0 and 1.0.
// If showProgress is false, fn is called directly without any UI.
func RunWithProgress[T any](label string, showProgress bool, fn func(update func(float64)) T) T {
	if !showProgress {
		return fn(func(float64) {})
	}

	m := progressModel[T]{
		label: label,
		bar:   progress.New(progress.WithSolidFill("#ff5300")),
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))

	go func() {
		result := fn(func(pct float64) {
			p.Send(progressMsg(pct))
		})
		p.Send(doneMsg[T]{result: result})
	}()

	finalModel, err := p.Run()
	if err != nil {
		// If bubbletea fails, fall back to running without progress
		return fn(func(float64) {})
	}

	return finalModel.(progressModel[T]).result
}
