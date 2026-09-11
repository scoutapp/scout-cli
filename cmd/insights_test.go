package cmd

import (
	"strings"
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatInsightValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"null renders as a dash", nil, "—"},
		{"empty string renders as a dash", "", "—"},
		{"string", "Controller/users/index", "Controller/users/index"},
		{"number", 42.5, "42.5"},
		{"bool", true, "true"},
		{"list is joined", []interface{}{"a", "b"}, "a, b"},
		{"empty list renders as a dash", []interface{}{}, "—"},
		{"empty map renders as a dash", map[string]interface{}{}, "—"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatInsightValue(tt.value))
		})
	}
}

// Fields arrive as a Go map, which ranges in a random order.
func TestPrintInsightFieldsIsSorted(t *testing.T) {
	fields := map[string]interface{}{
		"uri":               "/users",
		"kind":              "n_plus_one",
		"num_queries":       31.0,
		"potential_savings": 12.5,
		"metric_name":       "Controller/users/index",
	}

	var runs []string
	for i := 0; i < 5; i++ {
		runs = append(runs, captureStdout(t, func() {
			printInsightFields(fields, "  ")
		}))
	}

	for _, out := range runs {
		assert.Equal(t, runs[0], out, "field order must be stable across runs")
	}

	out := runs[0]
	for _, key := range []string{"kind", "metric_name", "num_queries", "potential_savings", "uri"} {
		assert.Contains(t, out, key+":")
	}
	assert.Less(t, strings.Index(out, "kind:"), strings.Index(out, "metric_name:"))
	assert.Less(t, strings.Index(out, "num_queries:"), strings.Index(out, "potential_savings:"))
	assert.Less(t, strings.Index(out, "potential_savings:"), strings.Index(out, "uri:"))
}

// Nested objects used to print as Go's map[...] syntax and nulls as <nil>.
func TestPrintInsightFieldsFormatsNestedValues(t *testing.T) {
	out := captureStdout(t, func() {
		printInsightFields(map[string]interface{}{
			"uri": nil,
			"raw_allocation_summary": map[string]interface{}{
				"ActiveRecord": map[string]interface{}{"call_count": 79.0},
			},
		}, "  ")
	})

	assert.NotContains(t, out, "map[")
	assert.NotContains(t, out, "<nil>")
	assert.Contains(t, out, "raw_allocation_summary:")
	assert.Contains(t, out, "ActiveRecord:")
	assert.Contains(t, out, "call_count:")
	assert.Contains(t, out, "uri:")
	assert.Contains(t, out, "—")

	// Nesting depth is reflected in the indentation.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var callCountLine string
	for _, l := range lines {
		if strings.Contains(l, "call_count:") {
			callCountLine = l
		}
	}
	require.NotEmpty(t, callCountLine)
	assert.True(t, strings.HasPrefix(callCountLine, "      "), "expected nested indent, got %q", callCountLine)
}

func TestLimitInsightCategories(t *testing.T) {
	items := []api.InsightItem{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	result := &api.InsightsListResult{
		Insights: map[string]api.InsightCategory{
			"n_plus_one": {Count: 3, NewCount: 1, Items: items},
			"slow_query": {Count: 0, NewCount: 0},
		},
	}

	withLimit(t, 2)
	limited := limitInsightCategories(result)

	assert.Len(t, limited.Insights["n_plus_one"].Items, 2)
	assert.Empty(t, limited.Insights["slow_query"].Items)
	// Counts describe what exists, so -n must not change them.
	assert.Equal(t, 3, limited.Insights["n_plus_one"].Count)
	assert.Equal(t, 1, limited.Insights["n_plus_one"].NewCount)
	// The original result is untouched.
	assert.Len(t, result.Insights["n_plus_one"].Items, 3)
}
