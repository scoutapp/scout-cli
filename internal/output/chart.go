package output

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
	"github.com/scoutapm/scout-cli/internal/api"
)

func RenderChart(title string, points []api.MetricPoint, summary float64, unit string) string {
	if len(points) == 0 {
		return DimStyle.Render("No data points available.")
	}

	values := make([]float64, len(points))
	var min, max, sum float64
	min = math.MaxFloat64
	for i, p := range points {
		values[i] = p.Value
		sum += p.Value
		if p.Value < min {
			min = p.Value
		}
		if p.Value > max {
			max = p.Value
		}
	}
	avg := sum / float64(len(values))

	// Downsample to ~60 points for terminal width
	if len(values) > 60 {
		values = downsample(values, 60)
	}

	graph := asciigraph.Plot(values,
		asciigraph.Height(8),
		asciigraph.Caption(fmt.Sprintf("  %s", title)),
	)

	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render(title))
	sb.WriteString("\n\n")
	sb.WriteString(graph)
	sb.WriteString("\n\n")

	statsStyle := lipgloss.NewStyle().Padding(0, 2)
	stats := fmt.Sprintf("Avg: %s%s  Min: %s%s  Max: %s%s",
		formatValue(avg), unit,
		formatValue(min), unit,
		formatValue(max), unit,
	)
	if summary > 0 {
		stats += fmt.Sprintf("  Summary: %s%s", formatValue(summary), unit)
	}
	sb.WriteString(statsStyle.Render(DimStyle.Render(stats)))

	return sb.String()
}

func formatValue(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fk", v/1000)
	}
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func downsample(data []float64, target int) []float64 {
	if len(data) <= target {
		return data
	}
	result := make([]float64, target)
	ratio := float64(len(data)) / float64(target)
	for i := 0; i < target; i++ {
		idx := int(float64(i) * ratio)
		if idx >= len(data) {
			idx = len(data) - 1
		}
		result[i] = data[idx]
	}
	return result
}
