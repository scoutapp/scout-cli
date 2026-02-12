package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/scoutapm/scout/internal/api"
)

func RenderSpanTree(trace api.TraceDetail) string {
	var sb strings.Builder

	totalMs := trace.DurationInSeconds * 1000
	if totalMs == 0 {
		totalMs = trace.TotalCallTime * 1000
	}

	// Header
	header := fmt.Sprintf("Trace #%d — %s — %s",
		trace.ID,
		trace.MetricName,
		FormatMs(totalMs),
	)
	sb.WriteString(HeaderStyle.Render(header))
	sb.WriteString("\n")

	if trace.LegacyFormat || len(trace.Spans) == 0 {
		sb.WriteString(DimStyle.Render("\n  Legacy trace format — no span tree available"))
		sb.WriteString("\n")
	} else {
		sb.WriteString("\n")
		for i, span := range trace.Spans {
			isLast := i == len(trace.Spans)-1
			renderSpan(&sb, span, totalMs, "", isLast)
		}
	}

	// Footer with metadata
	sb.WriteString("\n")
	var meta []string
	if trace.MemDelta != 0 {
		meta = append(meta, fmt.Sprintf("Memory: %s%s",
			sign(trace.MemDelta), FormatBytes(abs(trace.MemDelta))))
	}
	if trace.AllocationsCount > 0 {
		meta = append(meta, fmt.Sprintf("Allocations: %s", FormatNumber(trace.AllocationsCount)))
	}
	if trace.Hostname != "" {
		meta = append(meta, fmt.Sprintf("Host: %s", trace.Hostname))
	}
	if trace.GitSHA != "" {
		meta = append(meta, fmt.Sprintf("SHA: %s", trace.GitSHA))
	}
	if len(meta) > 0 {
		sb.WriteString(" " + DimStyle.Render(strings.Join(meta, "    ")))
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderSpan(sb *strings.Builder, span api.TraceSpan, totalMs float64, prefix string, isLast bool) {
	connector := " ├── "
	childPrefix := " │   "
	if isLast {
		connector = " └── "
		childPrefix = "     "
	}

	// Duration bar
	barWidth := 20
	ratio := span.DurationMs / totalMs
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(barWidth))
	if filled < 1 && span.DurationMs > 0 {
		filled = 1
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Color based on percentage of total
	barColor := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	if ratio > 0.8 {
		barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
	} else if ratio > 0.5 {
		barColor = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	}

	warning := ""
	if ratio > 0.5 {
		warning = WarningStyle.Render(" ⚠ slow")
	}

	// Format the operation name (pad to 40 chars)
	op := span.Operation
	if len(op) > 40 {
		op = op[:37] + "..."
	}
	opFormatted := fmt.Sprintf("%-40s", op)

	line := fmt.Sprintf("%s%s%s %6s  %s%s",
		prefix,
		connector,
		opFormatted,
		FormatMs(span.DurationMs),
		barColor.Render(bar),
		warning,
	)
	sb.WriteString(line)
	sb.WriteString("\n")

	// Render children
	for i, child := range span.Children {
		childIsLast := i == len(span.Children)-1
		renderSpan(sb, child, totalMs, prefix+childPrefix, childIsLast)
	}
}

func sign(n int64) string {
	if n >= 0 {
		return "+"
	}
	return "-"
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
