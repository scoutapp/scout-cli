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
