package output

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// series builds points spaced one minute apart, ending `endsAgo` before now.
func series(endsAgo time.Duration, values ...float64) []api.MetricPoint {
	end := time.Now().UTC().Add(-endsAgo)
	points := make([]api.MetricPoint, len(values))
	for i, v := range values {
		ts := end.Add(-time.Duration(len(values)-1-i) * time.Minute)
		points[i] = api.MetricPoint{Timestamp: ts.Format(time.RFC3339), Value: v}
	}
	return points
}

func TestDownsamplePreservesSpike(t *testing.T) {
	data := make([]float64, 600)
	for i := range data {
		data[i] = 10
	}
	data[317] = 999

	got := downsample(data, 60)

	require.Len(t, got, 60)
	assert.Contains(t, got, 999.0, "downsampling dropped the spike")
}

func TestDownsampleShortSeriesUnchanged(t *testing.T) {
	data := []float64{1, 2, 3}
	assert.Equal(t, data, downsample(data, 60))
}

func TestDownsampleCoversEveryBucket(t *testing.T) {
	data := make([]float64, 121)
	for i := range data {
		data[i] = float64(i)
	}

	got := downsample(data, 60)

	require.Len(t, got, 60)
	// Buckets are contiguous and ascending, so the last bucket must contain
	// the final value rather than stopping short of it.
	assert.Equal(t, 120.0, got[59])
	assert.Less(t, got[0], got[59])
}

func TestTrimPartialBucketDropsUnclosedBucket(t *testing.T) {
	now := time.Now().UTC()
	points := series(30*time.Second, 100, 110, 120, 0)

	got := trimPartialBucket(points, now)

	assert.Len(t, got, 3)
	assert.Equal(t, 120.0, got[len(got)-1].Value)
}

func TestTrimPartialBucketKeepsClosedBucket(t *testing.T) {
	now := time.Now().UTC()
	points := series(2*time.Hour, 100, 110, 120, 0)

	got := trimPartialBucket(points, now)

	assert.Len(t, got, 4)
}

func TestTrimPartialBucketIgnoresUnparsableTimestamps(t *testing.T) {
	points := []api.MetricPoint{
		{Timestamp: "not-a-time", Value: 1},
		{Timestamp: "also-not-a-time", Value: 2},
		{Timestamp: "still-not-a-time", Value: 3},
	}

	assert.Equal(t, points, trimPartialBucket(points, time.Now().UTC()))
}

func TestRenderChartExcludesPartialBucketFromStats(t *testing.T) {
	out := RenderChart("response_time", series(30*time.Second, 100, 110, 120, 0), 110, "ms")

	assert.Contains(t, out, "Min: 100ms")
	assert.Contains(t, out, "Max: 120ms")
	assert.NotContains(t, out, "Min: 0ms")
}

func TestRenderChartPromotesMillisecondsToSeconds(t *testing.T) {
	out := RenderChart("response_time", series(2*time.Hour, 5700, 5700, 5700), 5700, "ms")

	assert.Contains(t, out, "5.7s")
	assert.NotContains(t, out, "kms")
}

func TestRenderChartKeepsUnitForNonMilliseconds(t *testing.T) {
	out := RenderChart("throughput", series(2*time.Hour, 5700, 5700, 5700), 5700, " rpm")

	assert.Contains(t, out, "5.7k rpm")
}

func TestRenderChartShowsZeroSummary(t *testing.T) {
	out := RenderChart("errors", series(2*time.Hour, 1, 2, 3), 0, "")

	assert.Contains(t, out, "Summary: 0")
}

func TestRenderChartRendersTitleOnce(t *testing.T) {
	title := "response_time — api/metrics/show"

	out := RenderChart(title, series(2*time.Hour, 1, 2, 3), 2, "ms")

	assert.Equal(t, 1, strings.Count(out, title), "title should be rendered once, not duplicated as a caption")
}

func TestRenderChartNoPoints(t *testing.T) {
	assert.Contains(t, RenderChart("errors", nil, 0, ""), "No data points available.")
}

func TestFormatStat(t *testing.T) {
	tests := []struct {
		value float64
		unit  string
		want  string
	}{
		{5700, "ms", "5.7s"},
		{999, "ms", "999ms"},
		{1000, "ms", "1.0s"},
		{0, "ms", "0ms"},
		{1500, " rpm", "1.5k rpm"},
		{0.5, "", "0.50"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%s", tt.value, tt.unit), func(t *testing.T) {
			assert.Equal(t, tt.want, formatStat(tt.value, tt.unit))
		})
	}
}
