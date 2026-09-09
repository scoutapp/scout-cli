package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The API reports mem_delta as a float (megabytes), e.g. 0.0 or 2.9296875.
// Regression test for https://github.com/scoutapp/scout-cli/issues/19.
func TestListTracesFloatMemDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6/endpoints/YXBpL21ldHJpY3Mvc2hvdw==/traces", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"header": {"status": {"code": 200, "message": "OK"}, "apiVersion": "0.1"},
			"results": {"traces": [
				{"id": 1, "time": "2026-09-06T17:15:00.000-07:00", "total_call_time": 83.48, "mem_delta": 2.9296875, "metric_name": "Controller/api/metrics/show", "uri": "/api/v0/apps/4241/metrics/response_time", "context": {}},
				{"id": 2, "time": "2026-09-06T17:16:00.000-07:00", "total_call_time": 1.5, "mem_delta": 0.0, "metric_name": "Controller/api/metrics/show", "uri": null, "context": {}},
				{"id": 3, "time": "2026-09-06T17:17:00.000-07:00", "total_call_time": 1.5, "mem_delta": null, "metric_name": "Controller/api/metrics/show", "uri": "/x", "context": {}},
				{"id": 4, "time": "2026-09-06T17:18:00.000-07:00", "total_call_time": 1.5, "mem_delta": 12, "metric_name": "Controller/api/metrics/show", "uri": "/x", "context": {}}
			]}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	traces, err := client.ListTraces(6, "YXBpL21ldHJpY3Mvc2hvdw==", "2026-09-03T00:00:00Z", "2026-09-09T00:00:00Z")
	require.NoError(t, err)
	require.Len(t, traces, 4)
	assert.InDelta(t, 2.9296875, traces[0].MemDelta, 1e-9)
	assert.Equal(t, 0.0, traces[1].MemDelta)
	assert.Equal(t, 0.0, traces[2].MemDelta, "null mem_delta should decode as zero")
	assert.Equal(t, 12.0, traces[3].MemDelta, "integer mem_delta should still decode")
}

func TestGetTraceFloatMemDelta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6/traces/1827043988", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"header": {"status": {"code": 200, "message": "OK"}, "apiVersion": "0.1"},
			"results": {"trace": {
				"id": 1827043988, "time": "2026-09-06T17:15:00.000-07:00", "total_call_time": 83483.85,
				"mem_delta": 2.9296875, "metric_name": "Controller/api/metrics/show", "uri": "/x", "context": {},
				"transaction_id": "abc", "hostname": "web-1", "git_sha": "deadbeef",
				"duration_in_seconds": 83.483853, "allocations_count": 15467, "limited": false,
				"spans": [{"id": "span-1", "parent_id": null, "operation": "Middleware/Summary", "type": "Middleware",
				           "description": null, "duration_ms": 83.48, "exclusive_duration_ms": 0.0038, "allocations": 15467, "children": []}]
			}}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	trace, err := client.GetTrace(6, 1827043988)
	require.NoError(t, err)
	assert.InDelta(t, 2.9296875, trace.MemDelta, 1e-9)
	assert.Equal(t, int64(15467), trace.AllocationsCount)
	require.Len(t, trace.Spans, 1)
	assert.Equal(t, int64(15467), trace.Spans[0].Allocations)
}
