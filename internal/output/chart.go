package output

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
	"github.com/scoutapm/scout/internal/api"
)

// chartPoints is the number of plotted points a series is downsampled to so
// the chart fits a normal terminal width.
const chartPoints = 60

func RenderChart(title string, points []api.MetricPoint, summary float64, unit string) string {
	points = trimPartialBucket(points, time.Now().UTC())
	if len(points) == 0 {
		return DimStyle.Render("No data points available.")
	}

	values := make([]float64, len(points))
	lo, hi, sum := math.MaxFloat64, -math.MaxFloat64, 0.0
	for i, p := range points {
		values[i] = p.Value
		sum += p.Value
		lo = math.Min(lo, p.Value)
		hi = math.Max(hi, p.Value)
	}
	avg := sum / float64(len(values))

	if len(values) > chartPoints {
		values = downsample(values, chartPoints)
	}

	graph := asciigraph.Plot(values, asciigraph.Height(8))

	var sb strings.Builder
	sb.WriteString(HeaderStyle.Render(title))
	sb.WriteString("\n\n")
	sb.WriteString(graph)
	sb.WriteString("\n\n")

	statsStyle := lipgloss.NewStyle().Padding(0, 2)
	stats := fmt.Sprintf("Avg: %s  Min: %s  Max: %s  Summary: %s",
		formatStat(avg, unit),
		formatStat(lo, unit),
		formatStat(hi, unit),
		formatStat(summary, unit),
	)
	sb.WriteString(statsStyle.Render(DimStyle.Render(stats)))

	return sb.String()
}

// trimPartialBucket drops the final point when the time bucket it starts is
// still filling. The API's most recent bucket is incomplete and usually reads
// near zero, which would otherwise drag Min and Avg toward zero and put a
// phantom dip at the right edge of the plot.
func trimPartialBucket(points []api.MetricPoint, now time.Time) []api.MetricPoint {
	if len(points) < 3 {
		return points
	}
	last, err := time.Parse(time.RFC3339, points[len(points)-1].Timestamp)
	if err != nil {
		return points
	}
	prev, err := time.Parse(time.RFC3339, points[len(points)-2].Timestamp)
	if err != nil {
		return points
	}
	interval := last.Sub(prev)
	if interval <= 0 {
		return points
	}
	if last.Add(interval).After(now) {
		return points[:len(points)-1]
	}
	return points
}

// formatStat renders a stat value with its unit, promoting large millisecond
// values to seconds so the k-scale suffix never reads "kms".
func formatStat(v float64, unit string) string {
	if unit == "ms" && math.Abs(v) >= 1000 {
		return fmt.Sprintf("%.1fs", v/1000)
	}
	return formatValue(v) + unit
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

// downsample reduces data to target points by keeping the largest-magnitude
// value of each bucket, so a spike that falls between bucket boundaries still
// shows up in the plotted line.
func downsample(data []float64, target int) []float64 {
	if target <= 0 || len(data) <= target {
		return data
	}
	result := make([]float64, target)
	for i := range result {
		start := i * len(data) / target
		end := (i + 1) * len(data) / target
		if end <= start {
			end = start + 1
		}
		peak := data[start]
		for _, v := range data[start+1 : end] {
			if math.Abs(v) > math.Abs(peak) {
				peak = v
			}
		}
		result[i] = peak
	}
	return result
}
