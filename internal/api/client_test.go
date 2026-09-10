package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListApps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-SCOUT-API"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header": map[string]interface{}{
				"status":     map[string]interface{}{"code": 200, "message": "OK"},
				"apiVersion": "0.1",
			},
			"results": map[string]interface{}{
				"apps": []map[string]interface{}{
					{"id": 1, "name": "Test App", "last_reported_at": "2026-02-12T19:00:00Z"},
					{"id": 2, "name": "App 2"},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	apps, err := client.ListApps()
	require.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Equal(t, "Test App", apps[0].Name)
	assert.Equal(t, 1, apps[0].ID)
}

func TestGetApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header": map[string]interface{}{
				"status":     map[string]interface{}{"code": 200, "message": "OK"},
				"apiVersion": "0.1",
			},
			"results": map[string]interface{}{
				"app": map[string]interface{}{"id": 6, "name": "Scout"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	app, err := client.GetApp(6)
	require.NoError(t, err)
	assert.Equal(t, "Scout", app.Name)
	assert.Equal(t, 6, app.ID)
}

func TestGetMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/apps/6/metrics/response_time", r.URL.Path)
		assert.Equal(t, "2026-02-12T16:00:00Z", r.URL.Query().Get("from"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header": map[string]interface{}{
				"status":     map[string]interface{}{"code": 200, "message": "OK"},
				"apiVersion": "0.1",
			},
			"results": map[string]interface{}{
				"summaries": map[string]interface{}{"response_time": 106.758},
				"series": map[string]interface{}{
					"response_time": []interface{}{
						[]interface{}{"2026-02-12T19:15:00Z", 76.51},
						[]interface{}{"2026-02-12T19:16:00Z", 117.70},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	metrics, err := client.GetMetrics(6, "response_time", "2026-02-12T16:00:00Z", "2026-02-12T19:00:00Z")
	require.NoError(t, err)
	assert.InDelta(t, 106.758, metrics.Summaries["response_time"], 0.001)
	assert.Len(t, metrics.Series["response_time"], 2)
	assert.Equal(t, "2026-02-12T19:15:00Z", metrics.Series["response_time"][0].Timestamp)
	assert.InDelta(t, 76.51, metrics.Series["response_time"][0].Value, 0.01)
}

func TestListEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Endpoints return a bare array in results
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header": map[string]interface{}{
				"status":     map[string]interface{}{"code": 200, "message": "OK"},
				"apiVersion": "0.1",
			},
			"results": []map[string]interface{}{
				{
					"name":                  "api/metrics/show",
					"percentage_of_total":   48.03,
					"response_time":         92.65,
					"throughput":            403.26,
					"max_allocations":       474271,
					"error_rate":            0,
					"formatted_method_name": "Api::MetricsController#show",
					"link":                  "/apps/6/endpoints/abc==",
					"95th_percentile":       396.10,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	endpoints, err := client.ListEndpoints(6, "2026-02-12T16:00:00Z", "2026-02-12T19:00:00Z")
	require.NoError(t, err)
	assert.Len(t, endpoints, 1)
	assert.Equal(t, "api/metrics/show", endpoints[0].Name)
	assert.InDelta(t, 92.65, endpoints[0].ResponseTime, 0.01)
	assert.InDelta(t, 403.26, endpoints[0].Throughput, 0.01)
	assert.InDelta(t, 396.10, endpoints[0].P95, 0.01)
}

func TestAPIError403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"header": map[string]interface{}{
				"status":     map[string]interface{}{"code": 403, "message": "Forbidden"},
				"apiVersion": "0.1",
			},
			"results": nil,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	_, err := client.ListApps()
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestNonJSONErrorResponse(t *testing.T) {
	// Simulates an endpoint that isn't deployed yet: the server returns a
	// plain-text 404 rather than the JSON envelope.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	_, err := client.ListAnomalyEvents(6, "all", "", "", "2026-02-12T16:00:00Z", "2026-02-12T19:00:00Z")
	require.Error(t, err)

	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError, got %T: %v", err, err)
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "Not Found")
	// The cryptic JSON parse error must not leak through.
	assert.NotContains(t, err.Error(), "invalid character")
}

func TestMetricPointJSON(t *testing.T) {
	data := `["2026-02-12T19:15:00Z", 76.51]`
	var mp MetricPoint
	err := json.Unmarshal([]byte(data), &mp)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-12T19:15:00Z", mp.Timestamp)
	assert.InDelta(t, 76.51, mp.Value, 0.01)

	// Roundtrip
	out, err := json.Marshal(mp)
	require.NoError(t, err)
	assert.Contains(t, string(out), "2026-02-12T19:15:00Z")
	assert.Contains(t, string(out), "76.51")
}

func TestGetOrgUsage(t *testing.T) {
	// Full payload shape of /api/v0/usage with every optional section present (synthetic values).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v0/usage", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-SCOUT-API"))
		_, _ = w.Write([]byte(`{"header":{"status":{"code":200,"message":"OK"},"apiVersion":"0.1"},"results":{"billing_period":{"start":"2026-03-01T00:00:00+00:00","end":"2026-04-01T00:00:00+00:00"},"pricing_style":"per node","apm":{"total_transactions":12345678},"nodes":{"active_count":4},"errors":{"count":42,"limit":1000000},"logs":{"bytes_used":5000000000,"limit_bytes":2000000000000}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	usage, err := client.GetOrgUsage()
	require.NoError(t, err)

	assert.Equal(t, "2026-03-01T00:00:00+00:00", usage.BillingPeriod.Start)
	assert.Equal(t, "2026-04-01T00:00:00+00:00", usage.BillingPeriod.End)
	assert.Equal(t, "per node", usage.PricingStyle)

	require.NotNil(t, usage.APM)
	assert.Equal(t, int64(12345678), usage.APM.TotalTransactions)
	assert.Nil(t, usage.APM.Limit, "per-node plans have no transaction limit")

	require.NotNil(t, usage.Nodes)
	assert.Equal(t, 4, usage.Nodes.ActiveCount)

	require.NotNil(t, usage.Errors)
	assert.Equal(t, int64(42), usage.Errors.Count)
	assert.Equal(t, int64(1000000), usage.Errors.Limit)

	require.NotNil(t, usage.Logs)
	assert.Equal(t, int64(5000000000), usage.Logs.BytesUsed)
	require.NotNil(t, usage.Logs.LimitBytes)
	assert.Equal(t, int64(2000000000000), *usage.Logs.LimitBytes)
}

func TestGetOrgUsageMinimal(t *testing.T) {
	// Sections are omitted when the org has no per-node pricing, errors
	// add-on, or logs integration, and apm.limit is absent without a plan cap.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"header":{"status":{"code":200,"message":"OK"},"apiVersion":"0.1"},"results":{"billing_period":{"start":"2026-09-01T00:00:00+00:00","end":"2026-10-01T00:00:00+00:00"},"pricing_style":"per transaction","apm":{"total_transactions":12345}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	usage, err := client.GetOrgUsage()
	require.NoError(t, err)

	require.NotNil(t, usage.APM)
	assert.Equal(t, int64(12345), usage.APM.TotalTransactions)
	assert.Nil(t, usage.APM.Limit)
	assert.Nil(t, usage.Nodes)
	assert.Nil(t, usage.Errors)
	assert.Nil(t, usage.Logs)
}

func TestGetOrgUsageWithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"header":{"status":{"code":200,"message":"OK"},"apiVersion":"0.1"},"results":{"billing_period":{"start":null,"end":null},"pricing_style":"per transaction","apm":{"total_transactions":900000,"limit":1000000}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	usage, err := client.GetOrgUsage()
	require.NoError(t, err)

	assert.Empty(t, usage.BillingPeriod.Start, "null billing dates decode as empty strings")
	assert.Empty(t, usage.BillingPeriod.End)
	require.NotNil(t, usage.APM.Limit)
	assert.Equal(t, int64(1000000), *usage.APM.Limit)
}
