package output

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	HeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	DimStyle     = lipgloss.NewStyle().Faint(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	BoldStyle    = lipgloss.NewStyle().Bold(true)

	TableHeaderStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	TableCellStyle   = lipgloss.NewStyle().Padding(0, 1)
)

func StatusColor(status string) lipgloss.Style {
	switch status {
	case "active", "open", "ok":
		return SuccessStyle
	case "resolved", "closed":
		return DimStyle
	case "ignored":
		return WarningStyle
	default:
		return lipgloss.NewStyle()
	}
}

func ErrorRateColor(rate float64) lipgloss.Style {
	pct := rate * 100
	if pct > 5 {
		return ErrorStyle
	}
	if pct > 1 {
		return WarningStyle
	}
	return SuccessStyle
}

func FormatMs(ms float64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.1fs", ms/1000)
	}
	return fmt.Sprintf("%.0fms", ms)
}

func FormatSeconds(s float64) string {
	return FormatMs(s * 1000)
}

func FormatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func FormatRPM(n float64) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk rpm", n/1000)
	}
	return fmt.Sprintf("%.0f rpm", n)
}

func FormatPercent(n float64) string {
	return fmt.Sprintf("%.1f%%", n*100)
}

func FormatBytes(bytes int64) string {
	b := float64(bytes)
	if b >= 1e9 {
		return fmt.Sprintf("%.1f GB", b/1e9)
	}
	if b >= 1e6 {
		return fmt.Sprintf("%.1f MB", b/1e6)
	}
	if b >= 1e3 {
		return fmt.Sprintf("%.1f KB", b/1e3)
	}
	return fmt.Sprintf("%d B", bytes)
}

func FormatRelativeTime(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(math.Floor(d.Hours() / 24))
		return fmt.Sprintf("%dd ago", days)
	}
}
