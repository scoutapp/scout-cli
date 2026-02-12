package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) get(path string, params map[string]string) (json.RawMessage, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, err
	}

	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-SCOUT-API", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Header.Status.Code >= 400 {
		return nil, &APIError{
			StatusCode: apiResp.Header.Status.Code,
			Message:    apiResp.Header.Status.Message,
		}
	}

	return apiResp.Results, nil
}

func (c *Client) ListApps() ([]App, error) {
	results, err := c.get("/api/v0/apps", nil)
	if err != nil {
		return nil, err
	}
	var r AppsResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return r.Apps, nil
}

func (c *Client) GetApp(appID int) (App, error) {
	results, err := c.get(fmt.Sprintf("/api/v0/apps/%d", appID), nil)
	if err != nil {
		return App{}, err
	}
	var r AppResult
	if err := json.Unmarshal(results, &r); err != nil {
		return App{}, err
	}
	return r.App, nil
}

func (c *Client) GetMetrics(appID int, metricType, from, to string) (*MetricsResult, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/metrics/%s", appID, metricType)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r MetricsResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) ListEndpoints(appID int, from, to string) ([]EndpointEntry, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/endpoints", appID)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	// Endpoints response is a bare array
	var endpoints []EndpointEntry
	if err := json.Unmarshal(results, &endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil
}

func (c *Client) GetEndpointMetrics(appID int, endpoint, metricType, from, to string) (*MetricsResult, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/endpoints/%s/metrics/%s", appID, endpoint, metricType)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r MetricsResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) ListTraces(appID int, endpointID, from, to string) ([]TraceEntry, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/endpoints/%s/traces", appID, endpointID)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r TracesResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return r.Traces, nil
}

func (c *Client) GetTrace(appID, traceID int) (TraceDetail, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/traces/%d", appID, traceID)
	results, err := c.get(path, nil)
	if err != nil {
		return TraceDetail{}, err
	}
	var r TraceDetailResult
	if err := json.Unmarshal(results, &r); err != nil {
		return TraceDetail{}, err
	}
	return r.Trace, nil
}

func (c *Client) ListErrorGroups(appID int, from, to string) ([]ErrorGroup, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/error_groups", appID)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r ErrorGroupsResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return r.ErrorGroups, nil
}

func (c *Client) GetErrorGroup(appID, errorID int) (ErrorGroup, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/error_groups/%d", appID, errorID)
	results, err := c.get(path, nil)
	if err != nil {
		return ErrorGroup{}, err
	}
	var r ErrorGroupResult
	if err := json.Unmarshal(results, &r); err != nil {
		return ErrorGroup{}, err
	}
	return r.ErrorGroup, nil
}

func (c *Client) ListErrorOccurrences(appID, errorID int) ([]ErrorOccurrence, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/error_groups/%d/errors", appID, errorID)
	results, err := c.get(path, nil)
	if err != nil {
		return nil, err
	}
	var r ErrorOccurrencesResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return r.Errors, nil
}

func (c *Client) ListInsights(appID int, from, to string) (*InsightsListResult, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/insights", appID)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r InsightsListResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) GetInsightsByType(appID int, insightType, from, to string) (*InsightsShowResult, error) {
	path := fmt.Sprintf("/api/v0/apps/%d/insights/%s", appID, insightType)
	results, err := c.get(path, map[string]string{"from": from, "to": to})
	if err != nil {
		return nil, err
	}
	var r InsightsShowResult
	if err := json.Unmarshal(results, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
