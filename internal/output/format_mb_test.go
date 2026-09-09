package output

import (
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestFormatMB(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0 MB"},
		{0.35546875, "0.4 MB"},
		{2.9296875, "2.9 MB"},
		{7.29296875, "7.3 MB"},
		{12, "12 MB"},
		{256.4, "256 MB"},
		{1024, "1.0 GB"},
		{1536, "1.5 GB"},
		{-2.5, "-2.5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatMB(tt.input))
		})
	}
}

func TestRenderSpanTreeMemoryFooter(t *testing.T) {
	trace := api.TraceDetail{
		ID:                1,
		MetricName:        "Controller/api/metrics/show",
		DurationInSeconds: 0.5,
		MemDelta:          2.9296875,
		Spans: []api.TraceSpan{
			{ID: "a", Operation: "Controller/api/metrics/show", DurationMs: 500},
		},
	}
	out := RenderSpanTree(trace)
	assert.Contains(t, out, "Memory: +2.9 MB")

	trace.MemDelta = -0.5
	out = RenderSpanTree(trace)
	assert.Contains(t, out, "Memory: -0.5 MB")

	trace.MemDelta = 0
	out = RenderSpanTree(trace)
	assert.NotContains(t, out, "Memory:")
}
