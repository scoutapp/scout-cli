package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func envelope(results string) string {
	return `{"header":{"status":{"code":200,"message":"OK"},"apiVersion":"0.1"},"results":` + results + `}`
}

func TestListJobs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6/jobs", r.URL.Path)
		assert.Equal(t, "2026-01-01T00:00:00Z", r.URL.Query().Get("from"))
		assert.Equal(t, "test-key", r.Header.Get("X-SCOUT-API"))
		// Jobs response is a bare array
		_, _ = w.Write([]byte(envelope(`[
			{"full_name":"default/MyWorker","name":"MyWorker","queue":"default","throughput":120.5,"execution_time":85.25,"time_consumed":0.6,"latency":1.5,"job_id":"ZGVmYXVsdC9NeVdvcmtlcg=="},
			{"full_name":"mailers/SendWelcomeEmailJob","name":"SendWelcomeEmailJob","queue":"mailers","throughput":3.2,"execution_time":640,"time_consumed":0.4,"latency":0.25,"job_id":"bWFpbGVycy9TZW5kV2VsY29tZUVtYWlsSm9i"}
		]`)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	jobs, err := client.ListJobs(6, "2026-01-01T00:00:00Z", "2026-01-07T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "default/MyWorker", jobs[0].FullName)
	assert.Equal(t, "MyWorker", jobs[0].Name)
	assert.Equal(t, "default", jobs[0].Queue)
	assert.Equal(t, "ZGVmYXVsdC9NeVdvcmtlcg==", jobs[0].JobID)
	assert.InDelta(t, 120.5, jobs[0].Throughput, 0.01)
	assert.InDelta(t, 85.25, jobs[0].ExecutionTime, 0.01)
	assert.InDelta(t, 0.6, jobs[0].TimeConsumed, 0.0001)
	assert.InDelta(t, 1.5, jobs[0].Latency, 0.001)
	assert.Equal(t, "mailers", jobs[1].Queue)
}

func TestGetJobMetrics(t *testing.T) {
	tests := []struct {
		name          string
		metricType    string
		results       string
		wantSummary   float64
		wantSeries    []string
		wantTotalLen  int
		wantTotalVals []float64
	}{
		{
			name:         "throughput flat array",
			metricType:   "throughput",
			results:      `{"summaries":{"throughput":120.5},"series":{"throughput":[["2026-01-01T00:00:00Z",100.25],["2026-01-01T01:00:00Z",140.75]]}}`,
			wantSummary:  120.5,
			wantSeries:   []string{"throughput"},
			wantTotalLen: 2,
			wantTotalVals: []float64{
				100.25, 140.75,
			},
		},
		{
			name:       "execution_time nested categories summed by timestamp",
			metricType: "execution_time",
			results: `{"summaries":{"execution_time":85.25},"series":{"execution_time":{
				"ActiveRecord":[["2026-01-01T00:00:00Z",30],["2026-01-01T01:00:00Z",20]],
				"Ruby":[["2026-01-01T01:00:00Z",5],["2026-01-01T00:00:00Z",70]]}}}`,
			wantSummary:   85.25,
			wantSeries:    []string{"ActiveRecord", "Ruby"},
			wantTotalLen:  2,
			wantTotalVals: []float64{100, 25},
		},
		{
			name:       "execution_time with API-provided total is not double counted",
			metricType: "execution_time",
			results: `{"summaries":{"execution_time":110},"series":{"execution_time":{
				"ActiveRecord":[["2026-01-01T00:00:00Z",30]],
				"Job":[["2026-01-01T00:00:00Z",80]],
				"total":[["2026-01-01T00:00:00Z",110]]}}}`,
			wantSummary:   110,
			wantSeries:    []string{"ActiveRecord", "Job", "total"},
			wantTotalLen:  1,
			wantTotalVals: []float64{110},
		},
		{
			name:          "latency single-key object",
			metricType:    "latency",
			results:       `{"summaries":{"latency":1500},"series":{"latency":{"Latency":[["2026-01-01T00:00:00Z",1450]]}}}`,
			wantSummary:   1500,
			wantSeries:    []string{"Latency"},
			wantTotalLen:  1,
			wantTotalVals: []float64{1450},
		},
		{
			name:          "errors object summary",
			metricType:    "errors",
			results:       `{"summaries":{"errors":{"Job/default/MyWorker":0.02}},"series":{"errors":{"default/MyWorker":[["2026-01-01T00:00:00Z",0.05],["2026-01-01T01:00:00Z",0.01]]}}}`,
			wantSummary:   0.02,
			wantSeries:    []string{"default/MyWorker"},
			wantTotalLen:  2,
			wantTotalVals: []float64{0.05, 0.01},
		},
		{
			name:          "allocations integer values",
			metricType:    "allocations",
			results:       `{"summaries":{"allocations":50000},"series":{"allocations":{"Job/MyWorker":[["2026-01-01T00:00:00Z",12345]]}}}`,
			wantSummary:   50000,
			wantSeries:    []string{"Job/MyWorker"},
			wantTotalLen:  1,
			wantTotalVals: []float64{12345},
		},
		{
			name:         "unknown job empty flat series",
			metricType:   "throughput",
			results:      `{"summaries":{"throughput":0.0},"series":{"throughput":[]}}`,
			wantSummary:  0,
			wantSeries:   nil,
			wantTotalLen: 0,
		},
		{
			name:         "empty series object",
			metricType:   "execution_time",
			results:      `{"summaries":{"execution_time":null},"series":{}}`,
			wantSummary:  0,
			wantSeries:   nil,
			wantTotalLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v0/apps/6/jobs/ZGVmYXVsdC9NeVdvcmtlcg==/metrics/"+tt.metricType, r.URL.Path)
				_, _ = w.Write([]byte(envelope(tt.results)))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-key")
			m, err := client.GetJobMetrics(6, "ZGVmYXVsdC9NeVdvcmtlcg==", tt.metricType, "2026-01-01T00:00:00Z", "2026-01-07T00:00:00Z")
			require.NoError(t, err)
			assert.InDelta(t, tt.wantSummary, m.Summary, 0.001)

			keys := make([]string, 0, len(m.Series))
			for k := range m.Series {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, tt.wantSeries, keys)

			total := m.Total()
			assert.Len(t, total, tt.wantTotalLen)
			for i, v := range tt.wantTotalVals {
				assert.InDelta(t, v, total[i].Value, 0.001, "total[%d]", i)
			}
			// Total is sorted by timestamp ascending
			for i := 1; i < len(total); i++ {
				assert.Less(t, total[i-1].Timestamp, total[i].Timestamp)
			}
		})
	}
}

func TestJobMetricsResultJSONRoundTrip(t *testing.T) {
	raw := `{"summaries":{"execution_time":100},"series":{"execution_time":{"ActiveRecord":[["2026-01-01T00:00:00Z",30]],"Ruby":[["2026-01-01T00:00:00Z",70]]}}}`
	var m JobMetricsResult
	require.NoError(t, json.Unmarshal([]byte(raw), &m))

	out, err := json.Marshal(m)
	require.NoError(t, err)
	// Marshals as the normalized struct, not the raw API shape.
	assert.Contains(t, string(out), `"summary":100`)
	assert.Contains(t, string(out), `"ActiveRecord"`)
	assert.Contains(t, string(out), `"Ruby"`)
	assert.NotContains(t, string(out), `"summaries"`)
}

func TestJobMetricsResultUnexpectedShape(t *testing.T) {
	var m JobMetricsResult
	err := json.Unmarshal([]byte(`{"summaries":{"x":"str"},"series":{}}`), &m)
	require.Error(t, err)

	err = json.Unmarshal([]byte(`{"summaries":{},"series":{"x":"str"}}`), &m)
	require.Error(t, err)
}

func TestListJobTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6/jobs/ZGVmYXVsdC9NeVdvcmtlcg==/traces", r.URL.Path)
		assert.Equal(t, "2026-01-07T00:00:00Z", r.URL.Query().Get("to"))
		_, _ = w.Write([]byte(envelope(`{"traces":[
			{"id":501,"time":"2026-01-05T11:45:00Z","duration":2500,"name":"MyWorker","queue":"default","metric_name":"Job/default/MyWorker","context":{"host":"web-1"}},
			{"id":502,"time":"2026-01-05T11:46:00Z","duration":1200.5,"name":"MyWorker","queue":"default","metric_name":"Job/default/MyWorker","context":null}
		]}`)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	traces, err := client.ListJobTraces(6, "ZGVmYXVsdC9NeVdvcmtlcg==", "2026-01-01T00:00:00Z", "2026-01-07T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, traces, 2)
	assert.Equal(t, 501, traces[0].ID)
	assert.Equal(t, "2026-01-05T11:45:00Z", traces[0].Time)
	assert.InDelta(t, 2500, traces[0].Duration, 0.01)
	assert.Equal(t, "MyWorker", traces[0].Name)
	assert.Equal(t, "default", traces[0].Queue)
	assert.Equal(t, "Job/default/MyWorker", traces[0].MetricName)
	assert.Equal(t, "web-1", traces[0].Context["host"])
	assert.InDelta(t, 1200.5, traces[1].Duration, 0.01)
}

func TestListJobTracesNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header":  map[string]interface{}{"status": map[string]interface{}{"code": 404, "message": "Job not found"}, "apiVersion": "0.1"},
			"results": map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.ListJobTraces(6, "bm9wZQ==", "2026-01-01T00:00:00Z", "2026-01-07T00:00:00Z")
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "Job not found")
}
