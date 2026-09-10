package output

import (
	"strings"
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
)

// spanLine returns the rendered line containing op, with ANSI styling stripped.
func spanLine(t *testing.T, out, op string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, op) {
			return line
		}
	}
	t.Fatalf("no rendered line for span %q in:\n%s", op, out)
	return ""
}

func TestRenderSpanTreeSpanDurationMatchesTraceDuration(t *testing.T) {
	trace := api.TraceDetail{
		ID:                3,
		MetricName:        "Controller/users/index",
		DurationInSeconds: 7.9,
		Spans: []api.TraceSpan{
			{ID: "a", Operation: "Middleware/Summary", DurationSeconds: 7.9},
		},
	}
	out := RenderSpanTree(trace)

	assert.Contains(t, out, "Trace #3 — Controller/users/index — 7.9s")
	assert.Contains(t, spanLine(t, out, "Middleware/Summary"), "7.9s")
}

func TestRenderSpanTreeBarReflectsShareOfTrace(t *testing.T) {
	trace := api.TraceDetail{
		ID:                4,
		MetricName:        "Controller/users/index",
		DurationInSeconds: 10,
		Spans: []api.TraceSpan{
			{ID: "big", Operation: "ActiveRecord/User/find", DurationSeconds: 6},
			{ID: "small", Operation: "View/users/_row", DurationSeconds: 0.5},
		},
	}
	out := RenderSpanTree(trace)

	big := spanLine(t, out, "ActiveRecord/User/find")
	assert.Contains(t, big, "⚠ slow")
	assert.Equal(t, 12, strings.Count(big, "█"))
	assert.Equal(t, 8, strings.Count(big, "░"))

	small := spanLine(t, out, "View/users/_row")
	assert.NotContains(t, small, "⚠ slow")
	assert.Equal(t, 1, strings.Count(small, "█"))
	assert.Equal(t, 19, strings.Count(small, "░"))
}
