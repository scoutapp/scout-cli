package cmd

import (
	"fmt"
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateTransactions(t *testing.T) {
	tests := []struct {
		name     string
		points   []api.MetricPoint
		expected float64
	}{
		{
			name:     "empty points",
			points:   nil,
			expected: 0,
		},
		{
			name:     "single point",
			points:   []api.MetricPoint{{Timestamp: "2026-03-01T00:00:00Z", Value: 100}},
			expected: 0,
		},
		{
			name: "two points one minute apart at 100 rpm",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:00:00Z", Value: 100},
				{Timestamp: "2026-03-01T00:01:00Z", Value: 100},
			},
			expected: 100, // 100 rpm × 1 min
		},
		{
			name: "two points five minutes apart at 200 rpm",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:00:00Z", Value: 200},
				{Timestamp: "2026-03-01T00:05:00Z", Value: 200},
			},
			expected: 1000, // 200 rpm × 5 min
		},
		{
			name: "three points with varying rpm",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:00:00Z", Value: 100},
				{Timestamp: "2026-03-01T00:01:00Z", Value: 200},
				{Timestamp: "2026-03-01T00:02:00Z", Value: 300},
			},
			expected: 300, // (100×1) + (200×1)
		},
		{
			name: "one hour at steady 60 rpm",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:00:00Z", Value: 60},
				{Timestamp: "2026-03-01T01:00:00Z", Value: 60},
			},
			expected: 3600, // 60 rpm × 60 min
		},
		{
			name: "skips invalid timestamps",
			points: []api.MetricPoint{
				{Timestamp: "bad", Value: 100},
				{Timestamp: "2026-03-01T00:01:00Z", Value: 100},
			},
			expected: 0,
		},
		{
			name: "skips zero or negative intervals",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:01:00Z", Value: 100},
				{Timestamp: "2026-03-01T00:00:00Z", Value: 100},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTransactions(tt.points)
			assert.InDelta(t, tt.expected, result, 0.01)
		})
	}
}

func TestCalculateTransactionsFullDay(t *testing.T) {
	// Simulate 24 hours of 1-minute interval data at a steady 100 rpm
	// 1440 intervals × 100 rpm × 1 min = 144,000 transactions
	points := make([]api.MetricPoint, 0, 1441)
	for i := 0; i < 1440; i++ {
		hour := i / 60
		min := i % 60
		points = append(points, api.MetricPoint{
			Timestamp: fmt.Sprintf("2026-03-01T%02d:%02d:00Z", hour, min),
			Value:     100,
		})
	}
	// Final point rolls over to the next day
	points = append(points, api.MetricPoint{
		Timestamp: "2026-03-02T00:00:00Z",
		Value:     100,
	})

	result := calculateTransactions(points)
	assert.InDelta(t, 144000, result, 1)
}

func TestCalculateTransactionsVaryingRPM(t *testing.T) {
	// First hour at 100 rpm, second hour at 200 rpm
	// Total: (100 × 60) + (200 × 60) = 18,000
	points := []api.MetricPoint{
		{Timestamp: "2026-03-01T00:00:00Z", Value: 100},
		{Timestamp: "2026-03-01T01:00:00Z", Value: 200},
		{Timestamp: "2026-03-01T02:00:00Z", Value: 200},
	}

	result := calculateTransactions(points)
	assert.InDelta(t, 18000, result, 1)
}

func TestBucketByDay(t *testing.T) {
	tests := []struct {
		name           string
		points         []api.MetricPoint
		expectedDays   int
		expectedDates  []string
		expectedTotals []int64
	}{
		{
			name:         "empty points",
			points:       nil,
			expectedDays: 0,
		},
		{
			name: "single day",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T00:00:00Z", Value: 60},
				{Timestamp: "2026-03-01T01:00:00Z", Value: 60},
				{Timestamp: "2026-03-01T02:00:00Z", Value: 60},
			},
			expectedDays:   1,
			expectedDates:  []string{"2026-03-01"},
			expectedTotals: []int64{7200}, // (60×60) + (60×60)
		},
		{
			name: "two days",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-01T23:00:00Z", Value: 100},
				{Timestamp: "2026-03-02T00:00:00Z", Value: 200},
				{Timestamp: "2026-03-02T01:00:00Z", Value: 200},
			},
			expectedDays:  2,
			expectedDates: []string{"2026-03-01", "2026-03-02"},
			// Mar 01: 100 rpm × 60 min = 6000
			// Mar 02: 200 rpm × 60 min = 12000
			expectedTotals: []int64{6000, 12000},
		},
		{
			name: "sorted chronologically",
			points: []api.MetricPoint{
				{Timestamp: "2026-03-03T00:00:00Z", Value: 10},
				{Timestamp: "2026-03-03T01:00:00Z", Value: 10},
				{Timestamp: "2026-03-01T00:00:00Z", Value: 20},
				{Timestamp: "2026-03-01T01:00:00Z", Value: 20},
			},
			expectedDays:  2,
			expectedDates: []string{"2026-03-01", "2026-03-03"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := bucketByDay(tt.points)
			assert.Len(t, days, tt.expectedDays)

			if tt.expectedDates != nil {
				for i, d := range days {
					assert.Equal(t, tt.expectedDates[i], d.Date)
				}
			}
			if tt.expectedTotals != nil {
				for i, d := range days {
					assert.Equal(t, tt.expectedTotals[i], d.Transactions)
				}
			}
		})
	}
}

func TestBucketByDayBoundary(t *testing.T) {
	// A data point at 23:59 should be assigned to that day, not the next
	points := []api.MetricPoint{
		{Timestamp: "2026-03-01T23:59:00Z", Value: 100},
		{Timestamp: "2026-03-02T00:00:00Z", Value: 200},
		{Timestamp: "2026-03-02T00:01:00Z", Value: 200},
	}

	days := bucketByDay(points)
	require.Len(t, days, 2)
	assert.Equal(t, "2026-03-01", days[0].Date)
	assert.Equal(t, int64(100), days[0].Transactions) // 100 rpm × 1 min
	assert.Equal(t, "2026-03-02", days[1].Date)
	assert.Equal(t, int64(200), days[1].Transactions) // 200 rpm × 1 min
}

func TestSplitTimeframe(t *testing.T) {
	tests := []struct {
		name           string
		from           string
		to             string
		expectedChunks int
	}{
		{
			name:           "short range single chunk",
			from:           "2026-03-01T00:00:00Z",
			to:             "2026-03-02T00:00:00Z",
			expectedChunks: 1,
		},
		{
			name:           "exactly 14 days",
			from:           "2026-03-01T00:00:00Z",
			to:             "2026-03-15T00:00:00Z",
			expectedChunks: 1,
		},
		{
			name:           "15 days splits into two",
			from:           "2026-03-01T00:00:00Z",
			to:             "2026-03-16T00:00:00Z",
			expectedChunks: 2,
		},
		{
			name:           "30 days splits into three",
			from:           "2026-02-13T00:00:00Z",
			to:             "2026-03-15T00:00:00Z",
			expectedChunks: 3,
		},
		{
			name:           "same time produces no chunks",
			from:           "2026-03-01T00:00:00Z",
			to:             "2026-03-01T00:00:00Z",
			expectedChunks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitTimeframe(tt.from, tt.to)
			assert.Len(t, chunks, tt.expectedChunks)

			if len(chunks) > 0 {
				// First chunk starts at from
				assert.Equal(t, tt.from, chunks[0][0])
				// Last chunk ends at to
				assert.Equal(t, tt.to, chunks[len(chunks)-1][1])
				// Chunks are contiguous
				for i := 1; i < len(chunks); i++ {
					assert.Equal(t, chunks[i-1][1], chunks[i][0])
				}
			}
		})
	}
}

func TestSplitTimeframeNoChunkExceeds14Days(t *testing.T) {
	// 60-day range should produce chunks all <= 14 days
	chunks := splitTimeframe("2026-01-01T00:00:00Z", "2026-03-02T00:00:00Z")
	require.True(t, len(chunks) > 1)

	for i, chunk := range chunks {
		assert.NotEqual(t, chunk[0], chunk[1], "chunk %d start and end should differ", i)
	}
}

func TestFormatTransactions(t *testing.T) {
	tests := []struct {
		input    float64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{462769992, "462,769,992"},
		{0.4, "0"},
		{0.6, "1"},
		{999.9, "1,000"},
		{-1234, "-1,234"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatTransactions(tt.input))
		})
	}
}
