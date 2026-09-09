package cmd

import (
	"testing"
	"time"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestRenderUsageBar(t *testing.T) {
	tests := []struct {
		name     string
		used     int64
		limit    int64
		contains string
	}{
		{name: "zero usage", used: 0, limit: 1000, contains: "0.0%"},
		{name: "half usage", used: 500, limit: 1000, contains: "50.0%"},
		{name: "full usage", used: 1000, limit: 1000, contains: "100.0%"},
		{name: "over limit", used: 1500, limit: 1000, contains: "over limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, renderUsageBar(tt.used, tt.limit), tt.contains)
		})
	}
}

func TestRenderUsageBarFillWidth(t *testing.T) {
	// At 50%, half the 20-char bar should be filled.
	assert.Contains(t, renderUsageBar(50, 100), "██████████░░░░░░░░░░")
	// Over the limit the bar is capped at full width.
	assert.Contains(t, renderUsageBar(300, 100), "████████████████████")
}

func TestFormatBillingDate(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2026-03-01T00:00:00Z", "Mar 01, 2026"},
		{"2026-09-06T00:00:00+00:00", "Sep 06, 2026"},
		{"2026-03-01", "Mar 01, 2026"},
		{"2026-12-25T15:30:00Z", "Dec 25, 2026"},
		{"not-a-date", "not-a-date"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBillingDate(tt.input))
		})
	}
}

func TestParseBillingTime(t *testing.T) {
	got, err := parseBillingTime("2026-09-06T00:00:00+00:00")
	assert.NoError(t, err)
	assert.Equal(t, "2026-09-06T00:00:00Z", got.UTC().Format(time.RFC3339))

	got, err = parseBillingTime("2026-09-06")
	assert.NoError(t, err)
	assert.Equal(t, "2026-09-06T00:00:00Z", got.UTC().Format(time.RFC3339))

	_, err = parseBillingTime("nope")
	assert.Error(t, err)
}

func TestDaysRemaining(t *testing.T) {
	future := time.Now().UTC().Add(10*24*time.Hour + time.Hour).Format(time.RFC3339)
	assert.Equal(t, 10, daysRemaining(future))

	past := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	assert.Equal(t, 0, daysRemaining(past))

	assert.Equal(t, -1, daysRemaining("not-a-date"))
}

func TestRenderBillingSummary(t *testing.T) {
	limit := int64(500000000)
	logLimit := int64(10736344498176)
	u := &api.OrgUsage{
		BillingPeriod: api.BillingPeriod{Start: "2026-09-06T00:00:00+00:00", End: "2026-10-05T00:00:00+00:00"},
		PricingStyle:  "per node",
		APM:           &api.APMUsage{TotalTransactions: 329476789, Limit: &limit},
		Nodes:         &api.NodesUsage{ActiveCount: 15},
		Errors:        &api.ErrorsUsage{Count: 1156, Limit: 1000000000},
		Logs:          &api.LogsUsage{BytesUsed: 189223223871, LimitBytes: &logLimit},
	}

	out := renderBillingSummary(u)
	assert.Contains(t, out, "Sep 06, 2026 → Oct 05, 2026")
	assert.Contains(t, out, "Pricing: per node")
	assert.Contains(t, out, "Total: 329,476,789")
	assert.Contains(t, out, "Limit: 500,000,000")
	assert.Contains(t, out, "65.9%")
	assert.Contains(t, out, "Active: 15")
	assert.Contains(t, out, "Count: 1,156")
	assert.Contains(t, out, "Used: 189.2 GB")
	assert.Contains(t, out, "Limit: 10.7 TB")
}

func TestRenderBillingSummaryMinimal(t *testing.T) {
	u := &api.OrgUsage{
		PricingStyle: "per transaction",
		APM:          &api.APMUsage{TotalTransactions: 42},
	}

	out := renderBillingSummary(u)
	assert.Contains(t, out, "not available")
	assert.Contains(t, out, "Total: 42")
	assert.NotContains(t, out, "Limit:")
	assert.NotContains(t, out, "Nodes")
	assert.NotContains(t, out, "Errors")
	assert.NotContains(t, out, "Logs")
}

func TestServerTotalLine(t *testing.T) {
	assert.Equal(t,
		"Billing period total (server, web + jobs): 329,476,789 transactions",
		serverTotalLine(&api.APMUsage{TotalTransactions: 329476789}))

	limit := int64(500000000)
	assert.Equal(t,
		"Billing period total (server, web + jobs): 1,000 transactions (limit: 500,000,000)",
		serverTotalLine(&api.APMUsage{TotalTransactions: 1000, Limit: &limit}))
}
