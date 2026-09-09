package cmd

import (
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
)

const (
	testJobFullName = "default/Checkin::TraceAnalysisJob"
	testJobID       = "ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i"
)

func TestResolveJobID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full name is encoded", testJobFullName, testJobID},
		{"encoded id passes through", testJobID, testJobID},
		{"full name needing padding", "default/PlanTransactionsEnforcementJob", "ZGVmYXVsdC9QbGFuVHJhbnNhY3Rpb25zRW5mb3JjZW1lbnRKb2I="},
		{"Job/ prefixed full name is encoded", "Job/default/MyWorker", "Sm9iL2RlZmF1bHQvTXlXb3JrZXI="},
		{"empty passes through", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveJobID(tt.input))
		})
	}
}

func TestDecodeJobID(t *testing.T) {
	name, ok := decodeJobID(testJobID)
	assert.True(t, ok)
	assert.Equal(t, testJobFullName, name)

	// Round trip
	assert.Equal(t, testJobID, resolveJobID(name))

	// Padded id
	name, ok = decodeJobID("ZGVmYXVsdC9QbGFuVHJhbnNhY3Rpb25zRW5mb3JjZW1lbnRKb2I=")
	assert.True(t, ok)
	assert.Equal(t, "default/PlanTransactionsEnforcementJob", name)

	// Unpadded id is tolerated
	name, ok = decodeJobID("ZGVmYXVsdC9QbGFuVHJhbnNhY3Rpb25zRW5mb3JjZW1lbnRKb2I")
	assert.True(t, ok)
	assert.Equal(t, "default/PlanTransactionsEnforcementJob", name)

	_, ok = decodeJobID("")
	assert.False(t, ok)

	_, ok = decodeJobID("not base64!!")
	assert.False(t, ok)
}

func TestJobDisplayName(t *testing.T) {
	assert.Equal(t, testJobFullName, jobDisplayName(testJobFullName))
	assert.Equal(t, testJobFullName, jobDisplayName(testJobID))
	assert.Equal(t, "???", jobDisplayName("???"))
}

func TestIsValidJobMetricType(t *testing.T) {
	for _, v := range []string{"throughput", "execution_time", "latency", "errors", "allocations"} {
		assert.True(t, isValidJobMetricType(v), v)
	}
	assert.False(t, isValidJobMetricType("response_time"))
	assert.False(t, isValidJobMetricType(""))
}

func TestUnitForJobMetricType(t *testing.T) {
	assert.Equal(t, "/min", unitForJobMetricType("throughput"))
	assert.Equal(t, "ms", unitForJobMetricType("execution_time"))
	assert.Equal(t, "ms", unitForJobMetricType("latency"))
	assert.Equal(t, "", unitForJobMetricType("errors"))
	assert.Equal(t, "", unitForJobMetricType("allocations"))
}

func TestFormatTimeConsumed(t *testing.T) {
	assert.Equal(t, "26.3%", formatTimeConsumed(0.2633695))
	assert.Equal(t, "0.0%", formatTimeConsumed(0.00000059))
	assert.Equal(t, "100.0%", formatTimeConsumed(1))
}

func TestChartSeries(t *testing.T) {
	latency := []api.MetricPoint{{Timestamp: "2026-09-03T22:00:00Z", Value: 2239.77}}
	execTotal := []api.MetricPoint{{Timestamp: "2026-09-03T22:00:00Z", Value: 107.2}}

	// latency: the "total" sub-series is execution time, so chart "Latency".
	m := &api.JobMetricsResult{Series: map[string][]api.MetricPoint{"Latency": latency, "total": execTotal}}
	assert.Equal(t, latency, chartSeries("latency", m))

	// execution_time: prefer the API-provided total.
	m = &api.JobMetricsResult{Series: map[string][]api.MetricPoint{"ActiveRecord": latency, "total": execTotal}}
	assert.Equal(t, execTotal, chartSeries("execution_time", m))

	// Single series falls through to Total().
	m = &api.JobMetricsResult{Series: map[string][]api.MetricPoint{"throughput": latency}}
	assert.Equal(t, latency, chartSeries("throughput", m))

	// latency without a "Latency" key falls back to Total().
	m = &api.JobMetricsResult{Series: map[string][]api.MetricPoint{"total": execTotal}}
	assert.Equal(t, execTotal, chartSeries("latency", m))

	// Empty result.
	assert.Nil(t, chartSeries("latency", &api.JobMetricsResult{}))
}
