package output

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// maxCellWidth caps how wide a single table cell may render. Without it one
// long URI or job name stretches the whole table past a normal terminal width
// and wraps every row. Structured output is uncapped, so scripts still get the
// full value.
const maxCellWidth = 60

func RenderTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return DimStyle.Render("No results found.")
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return TableHeaderStyle
			}
			return TableCellStyle
		})

	for _, row := range rows {
		capped := make([]string, len(row))
		for i, cell := range row {
			capped[i] = Truncate(cell, maxCellWidth)
		}
		t.Row(capped...)
	}

	return fmt.Sprint(t)
}
