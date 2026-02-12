package api

import (
	"encoding/json"
	"fmt"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Header  ResponseHeader  `json:"header"`
	Results json.RawMessage `json:"results"`
}

type ResponseHeader struct {
	Status     ResponseStatus `json:"status"`
	APIVersion string         `json:"apiVersion"`
}

type ResponseStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// App represents a Scout application.
type App struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	LastReportedAt string `json:"last_reported_at,omitempty"`
}

// AppsResult wraps the apps list response.
type AppsResult struct {
	Apps []App `json:"apps"`
}

// AppResult wraps a single app response.
type AppResult struct {
	App App `json:"app"`
}

// MetricPoint is a [timestamp, value] tuple from the API.
type MetricPoint struct {
	Timestamp string
	Value     float64
}

func (mp *MetricPoint) UnmarshalJSON(data []byte) error {
	var raw [2]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[0], &mp.Timestamp); err != nil {
		return err
	}
	if err := json.Unmarshal(raw[1], &mp.Value); err != nil {
		return err
	}
	return nil
}

func (mp MetricPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]interface{}{mp.Timestamp, mp.Value})
}

// MetricsResult contains summaries and time series data.
type MetricsResult struct {
	Summaries map[string]float64        `json:"summaries"`
	Series    map[string][]MetricPoint  `json:"series"`
}

// EndpointEntry represents a single endpoint's performance data.
type EndpointEntry struct {
	Name                string  `json:"name"`
	PercentageOfTotal   float64 `json:"percentage_of_total"`
	ResponseTime        float64 `json:"response_time"`
	Throughput          float64 `json:"throughput"`
	MaxAllocations      float64 `json:"max_allocations"`
	ErrorRate           float64 `json:"error_rate"`
	FormattedMethodName string  `json:"formatted_method_name"`
	Link                string  `json:"link"`
	P95                 float64 `json:"95th_percentile"`
}

// TraceEntry represents a trace in a list.
type TraceEntry struct {
	ID            int                    `json:"id"`
	Time          string                 `json:"time"`
	TotalCallTime float64                `json:"total_call_time"`
	MemDelta      int64                  `json:"mem_delta"`
	MetricName    string                 `json:"metric_name"`
	URI           string                 `json:"uri"`
	Context       map[string]interface{} `json:"context"`
}

// TracesResult wraps the traces list response.
type TracesResult struct {
	Traces []TraceEntry `json:"traces"`
}

// TraceSpan represents a span in a trace tree.
type TraceSpan struct {
	ID                  string      `json:"id"`
	ParentID            *string     `json:"parent_id"`
	Operation           string      `json:"operation"`
	Type                string      `json:"type"`
	Description         *string     `json:"description"`
	DurationMs          float64     `json:"duration_ms"`
	ExclusiveDurationMs float64     `json:"exclusive_duration_ms"`
	Allocations         int64       `json:"allocations"`
	Children            []TraceSpan `json:"children,omitempty"`
}

// TraceDetail contains full trace information including spans.
type TraceDetail struct {
	ID                int         `json:"id"`
	Time              string      `json:"time"`
	TotalCallTime     float64     `json:"total_call_time"`
	MemDelta          int64       `json:"mem_delta"`
	MetricName        string      `json:"metric_name"`
	URI               string      `json:"uri"`
	TransactionID     string      `json:"transaction_id"`
	Hostname          string      `json:"hostname"`
	GitSHA            string      `json:"git_sha"`
	DurationInSeconds float64     `json:"duration_in_seconds"`
	AllocationsCount  int64       `json:"allocations_count"`
	Limited           bool        `json:"limited"`
	LegacyFormat      bool        `json:"legacy_format"`
	Spans             []TraceSpan `json:"spans"`
}

// TraceDetailResult wraps a single trace response.
type TraceDetailResult struct {
	Trace TraceDetail `json:"trace"`
}

// ErrorGroup represents an error group.
type ErrorGroup struct {
	ID                int                    `json:"id"`
	Name              string                 `json:"name"`
	Message           string                 `json:"message"`
	Status            string                 `json:"status"`
	ErrorsCount       int                    `json:"errors_count"`
	LastErrorAt       string                 `json:"last_error_at"`
	RequestComponents json.RawMessage         `json:"request_components"`
	RequestURI        string                 `json:"request_uri"`
	AppEnvironment    string                 `json:"app_environment"`
	LatestError       *ErrorOccurrence       `json:"latest_error,omitempty"`
}

// ErrorGroupsResult wraps the error groups list response.
type ErrorGroupsResult struct {
	ErrorGroups []ErrorGroup `json:"error_groups"`
}

// ErrorGroupResult wraps a single error group response.
type ErrorGroupResult struct {
	ErrorGroup ErrorGroup `json:"error_group"`
}

// ErrorOccurrence represents an individual error occurrence.
type ErrorOccurrence struct {
	ID             int                    `json:"id"`
	Name           string                 `json:"name"`
	Message        string                 `json:"message"`
	CreatedAt      string                 `json:"created_at"`
	Location       string                 `json:"location"`
	RequestURI     string                 `json:"request_uri"`
	RequestParams  map[string]interface{} `json:"request_params"`
	RequestSession map[string]interface{} `json:"request_session"`
	Context        map[string]interface{} `json:"context"`
	Trace          []string               `json:"trace"`
}

// ErrorOccurrencesResult wraps the error occurrences list response.
type ErrorOccurrencesResult struct {
	Errors []ErrorOccurrence `json:"errors"`
}

// InsightItem represents a single insight.
type InsightItem struct {
	ID     int                    `json:"id"`
	Name   string                 `json:"name"`
	Fields map[string]interface{} `json:"fields"`
}

// InsightCategory contains items for one insight type.
type InsightCategory struct {
	Count    int           `json:"count"`
	NewCount int           `json:"new_count"`
	Items    []InsightItem `json:"items"`
}

// InsightsTimeframe describes the time window for insights.
type InsightsTimeframe struct {
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	DurationMinutes float64 `json:"duration_minutes"`
}

// InsightsListResult wraps the insights list response.
type InsightsListResult struct {
	Timeframe InsightsTimeframe          `json:"timeframe"`
	Insights  map[string]InsightCategory `json:"insights"`
}

// InsightsShowResult wraps the insights show by type response.
type InsightsShowResult struct {
	Timeframe   InsightsTimeframe `json:"timeframe"`
	InsightType string            `json:"insight_type"`
	TotalCount  int               `json:"total_count"`
	NewCount    int               `json:"new_count"`
	Items       []InsightItem     `json:"items"`
}

// APIError represents an error from the Scout API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}
