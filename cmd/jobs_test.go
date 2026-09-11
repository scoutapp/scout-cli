package cmd

import (
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJobFullName = "default/MyWorker"
	testJobID       = "ZGVmYXVsdC9NeVdvcmtlcg=="
)

func TestResolveJobID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full name is encoded", testJobFullName, testJobID},
		{"encoded id passes through", testJobID, testJobID},
		{"full name needing padding", "default/ReportJob", "ZGVmYXVsdC9SZXBvcnRKb2I="},
		{"Job/ prefixed full name is encoded", "Job/default/MyWorker", "Sm9iL2RlZmF1bHQvTXlXb3JrZXI="},
		{"unpadded id passes through", "ZGVmYXVsdC9SZXBvcnRKb2I", "ZGVmYXVsdC9SZXBvcnRKb2I"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveJobID(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveJobIDRejectsMalformed(t *testing.T) {
	// A job class name copy-pasted without its queue, a value that isn't
	// base64 at all, and base64 that doesn't decode to a "queue/JobName".
	for _, input := range []string{"MyWorker", "SomeWorkerName", "not base64!!", "", "YWJj"} {
		_, err := resolveJobID(input)
		require.Error(t, err, input)
		assert.Contains(t, err.Error(), "default/MyWorker", input)
	}
}

func TestDecodeBase64ID(t *testing.T) {
	name, ok := decodeBase64ID(testJobID)
	assert.True(t, ok)
	assert.Equal(t, testJobFullName, name)

	// Round trip
	encoded, err := resolveJobID(name)
	require.NoError(t, err)
	assert.Equal(t, testJobID, encoded)

	// Padded id
	name, ok = decodeBase64ID("ZGVmYXVsdC9SZXBvcnRKb2I=")
	assert.True(t, ok)
	assert.Equal(t, "default/ReportJob", name)

	// Unpadded id is tolerated
	name, ok = decodeBase64ID("ZGVmYXVsdC9SZXBvcnRKb2I")
	assert.True(t, ok)
	assert.Equal(t, "default/ReportJob", name)

	_, ok = decodeBase64ID("")
	assert.False(t, ok)

	_, ok = decodeBase64ID("not base64!!")
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
	assert.Equal(t, "60.0%", formatTimeConsumed(0.6))
	assert.Equal(t, "0.0%", formatTimeConsumed(0.00000059))
	assert.Equal(t, "100.0%", formatTimeConsumed(1))
}

func TestChartSeries(t *testing.T) {
	latency := []api.MetricPoint{{Timestamp: "2026-01-01T00:00:00Z", Value: 1500}}
	execTotal := []api.MetricPoint{{Timestamp: "2026-01-01T00:00:00Z", Value: 85.25}}

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
