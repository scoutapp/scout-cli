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
		assert.Equal(t, "2026-09-03T00:00:00Z", r.URL.Query().Get("from"))
		assert.Equal(t, "test-key", r.Header.Get("X-SCOUT-API"))
		// Jobs response is a bare array
		_, _ = w.Write([]byte(envelope(`[
			{"full_name":"default/Checkin::TraceAnalysisJob","name":"Checkin::TraceAnalysisJob","queue":"default","throughput":1302.9635912698413,"execution_time":110.66311327207139,"time_consumed":0.2633695370502584,"latency":2.5040768210511826,"job_id":"ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i"},
			{"full_name":"mailers/DigestEmail::DeliverJob","name":"DigestEmail::DeliverJob","queue":"mailers","throughput":2.1,"execution_time":828.4,"time_consumed":0.00288,"latency":1.217,"job_id":"bWFpbGVycy9EaWdlc3RFbWFpbDo6RGVsaXZlckpvYg=="}
		]`)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	jobs, err := client.ListJobs(6, "2026-09-03T00:00:00Z", "2026-09-09T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "default/Checkin::TraceAnalysisJob", jobs[0].FullName)
	assert.Equal(t, "Checkin::TraceAnalysisJob", jobs[0].Name)
	assert.Equal(t, "default", jobs[0].Queue)
	assert.Equal(t, "ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i", jobs[0].JobID)
	assert.InDelta(t, 1302.96, jobs[0].Throughput, 0.01)
	assert.InDelta(t, 110.66, jobs[0].ExecutionTime, 0.01)
	assert.InDelta(t, 0.2634, jobs[0].TimeConsumed, 0.0001)
	assert.InDelta(t, 2.504, jobs[0].Latency, 0.001)
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
			results:      `{"summaries":{"throughput":1294.74},"series":{"throughput":[["2026-09-03T22:00:00Z",1059.45],["2026-09-03T23:00:00Z",1282.5]]}}`,
			wantSummary:  1294.74,
			wantSeries:   []string{"throughput"},
			wantTotalLen: 2,
			wantTotalVals: []float64{
				1059.45, 1282.5,
			},
		},
		{
			name:       "execution_time nested categories summed by timestamp",
			metricType: "execution_time",
			results: `{"summaries":{"execution_time":110.96},"series":{"execution_time":{
				"ActiveRecord":[["2026-09-03T22:00:00Z",30],["2026-09-03T23:00:00Z",20]],
				"Ruby":[["2026-09-03T23:00:00Z",5],["2026-09-03T22:00:00Z",70]]}}}`,
			wantSummary:   110.96,
			wantSeries:    []string{"ActiveRecord", "Ruby"},
			wantTotalLen:  2,
			wantTotalVals: []float64{100, 25},
		},
		{
			name:       "execution_time with API-provided total is not double counted",
			metricType: "execution_time",
			results: `{"summaries":{"execution_time":110},"series":{"execution_time":{
				"ActiveRecord":[["2026-09-03T22:00:00Z",30]],
				"Job":[["2026-09-03T22:00:00Z",80]],
				"total":[["2026-09-03T22:00:00Z",110]]}}}`,
			wantSummary:   110,
			wantSeries:    []string{"ActiveRecord", "Job", "total"},
			wantTotalLen:  1,
			wantTotalVals: []float64{110},
		},
		{
			name:          "latency single-key object",
			metricType:    "latency",
			results:       `{"summaries":{"latency":2568.46},"series":{"latency":{"Latency":[["2026-09-03T22:00:00Z",2239.77]]}}}`,
			wantSummary:   2568.46,
			wantSeries:    []string{"Latency"},
			wantTotalLen:  1,
			wantTotalVals: []float64{2239.77},
		},
		{
			name:          "errors object summary",
			metricType:    "errors",
			results:       `{"summaries":{"errors":{"Job/default/Checkin::TraceAnalysisJob":0.026}},"series":{"errors":{"default/Checkin::TraceAnalysisJob":[["2026-09-03T22:00:00Z",0.05],["2026-09-03T23:00:00Z",0.0166]]}}}`,
			wantSummary:   0.026,
			wantSeries:    []string{"default/Checkin::TraceAnalysisJob"},
			wantTotalLen:  2,
			wantTotalVals: []float64{0.05, 0.0166},
		},
		{
			name:          "allocations integer values",
			metricType:    "allocations",
			results:       `{"summaries":{"allocations":821813},"series":{"allocations":{"Job/Checkin::TraceAnalysisJob":[["2026-09-03T22:00:00Z",180764]]}}}`,
			wantSummary:   821813,
			wantSeries:    []string{"Job/Checkin::TraceAnalysisJob"},
			wantTotalLen:  1,
			wantTotalVals: []float64{180764},
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
				assert.Equal(t, "/api/v0/apps/6/jobs/ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i/metrics/"+tt.metricType, r.URL.Path)
				_, _ = w.Write([]byte(envelope(tt.results)))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-key")
			m, err := client.GetJobMetrics(6, "ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i", tt.metricType, "2026-09-03T00:00:00Z", "2026-09-09T00:00:00Z")
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
	raw := `{"summaries":{"execution_time":100},"series":{"execution_time":{"ActiveRecord":[["2026-09-03T22:00:00Z",30]],"Ruby":[["2026-09-03T22:00:00Z",70]]}}}`
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
		assert.Equal(t, "/api/v0/apps/6/jobs/ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i/traces", r.URL.Path)
		assert.Equal(t, "2026-09-09T00:00:00Z", r.URL.Query().Get("to"))
		_, _ = w.Write([]byte(envelope(`{"traces":[
			{"id":76462021,"time":"2026-09-08T11:45:00-04:00","duration":23271,"name":"Checkin::TraceAnalysisJob","queue":"default","metric_name":"Job/default/Checkin::TraceAnalysisJob","context":{"host":"web-1"}},
			{"id":76462022,"time":"2026-09-08T11:46:00-04:00","duration":1200.5,"name":"Checkin::TraceAnalysisJob","queue":"default","metric_name":"Job/default/Checkin::TraceAnalysisJob","context":null}
		]}`)))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	traces, err := client.ListJobTraces(6, "ZGVmYXVsdC9DaGVja2luOjpUcmFjZUFuYWx5c2lzSm9i", "2026-09-03T00:00:00Z", "2026-09-09T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, traces, 2)
	assert.Equal(t, 76462021, traces[0].ID)
	assert.Equal(t, "2026-09-08T11:45:00-04:00", traces[0].Time)
	assert.InDelta(t, 23271, traces[0].Duration, 0.01)
	assert.Equal(t, "Checkin::TraceAnalysisJob", traces[0].Name)
	assert.Equal(t, "default", traces[0].Queue)
	assert.Equal(t, "Job/default/Checkin::TraceAnalysisJob", traces[0].MetricName)
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
	_, err := client.ListJobTraces(6, "bm9wZQ==", "2026-09-03T00:00:00Z", "2026-09-09T00:00:00Z")
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "Job not found")
}
