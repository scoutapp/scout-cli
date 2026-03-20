package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderUsageBar(t *testing.T) {
	tests := []struct {
		name     string
		used     int64
		limit    int64
		contains string
	}{
		{
			name:     "zero usage",
			used:     0,
			limit:    1000,
			contains: "0.0%",
		},
		{
			name:     "half usage",
			used:     500,
			limit:    1000,
			contains: "50.0%",
		},
		{
			name:     "full usage",
			used:     1000,
			limit:    1000,
			contains: "100.0%",
		},
		{
			name:     "over limit",
			used:     1500,
			limit:    1000,
			contains: "over limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderUsageBar(tt.used, tt.limit)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestRenderUsageBarFillWidth(t *testing.T) {
	// At 50%, roughly half the bar should be filled
	bar := renderUsageBar(50, 100)
	assert.Contains(t, bar, "██████████░░░░░░░░░░")
}

func TestFormatBillingDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-03-01T00:00:00Z", "Mar 01, 2026"},
		{"2026-03-01", "Mar 01, 2026"},
		{"2026-12-25T15:30:00Z", "Dec 25, 2026"},
		{"not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBillingDate(tt.input))
		})
	}
}
