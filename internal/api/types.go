package api

import (
	"encoding/json"
	"fmt"
	"sort"
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
	Summaries map[string]float64       `json:"summaries"`
	Series    map[string][]MetricPoint `json:"series"`
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

// JobEntry represents a background job's performance data in a list.
type JobEntry struct {
	// FullName is "queue/JobName"; JobID is its Base64 URL-safe encoding.
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Queue    string `json:"queue"`
	JobID    string `json:"job_id"`
	// Throughput is jobs per minute.
	Throughput float64 `json:"throughput"`
	// ExecutionTime is the average execution time in milliseconds.
	ExecutionTime float64 `json:"execution_time"`
	// TimeConsumed is the fraction (0–1) of total job time consumed by this job.
	TimeConsumed float64 `json:"time_consumed"`
	// Latency is the average queue latency in seconds.
	Latency float64 `json:"latency"`
}

// JobTraceEntry represents a background job trace in a list.
type JobTraceEntry struct {
	ID   int    `json:"id"`
	Time string `json:"time"`
	// Duration is the execution duration in milliseconds.
	Duration   float64                `json:"duration"`
	Name       string                 `json:"name"`
	Queue      string                 `json:"queue"`
	MetricName string                 `json:"metric_name"`
	Context    map[string]interface{} `json:"context"`
}

// JobTracesResult wraps the job traces list response.
type JobTracesResult struct {
	Traces []JobTraceEntry `json:"traces"`
}

// JobMetricsResult contains a job metric summary and its time series.
//
// The API returns different shapes per metric type: the summary is either a
// number or an object of numbers, and the series is either a flat array of
// points or an object keyed by category (e.g. "ActiveRecord", "Ruby"). This
// type normalizes both into a single Summary and a map of named sub-series.
type JobMetricsResult struct {
	Summary float64                  `json:"summary"`
	Series  map[string][]MetricPoint `json:"series"`
}

func (r *JobMetricsResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		Summaries map[string]json.RawMessage `json:"summaries"`
		Series    map[string]json.RawMessage `json:"series"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Summary = 0
	r.Series = map[string][]MetricPoint{}

	for _, sum := range raw.Summaries {
		if isJSONNull(sum) {
			continue
		}
		var n float64
		if err := json.Unmarshal(sum, &n); err == nil {
			r.Summary += n
			continue
		}
		var m map[string]float64
		if err := json.Unmarshal(sum, &m); err != nil {
			return fmt.Errorf("job metrics: unexpected summary shape: %w", err)
		}
		for _, v := range m {
			r.Summary += v
		}
	}

	for metricType, series := range raw.Series {
		if isJSONNull(series) {
			continue
		}
		var flat []MetricPoint
		if err := json.Unmarshal(series, &flat); err == nil {
			if len(flat) > 0 {
				r.Series[metricType] = flat
			}
			continue
		}
		var nested map[string][]MetricPoint
		if err := json.Unmarshal(series, &nested); err != nil {
			return fmt.Errorf("job metrics: unexpected series shape: %w", err)
		}
		for category, pts := range nested {
			if len(pts) > 0 {
				r.Series[category] = pts
			}
		}
	}
	return nil
}

// Total returns a single series for charting: the only sub-series when there
// is exactly one, the API-provided "total" sub-series when present (e.g.
// execution_time returns per-category series plus their total), otherwise the
// per-timestamp sum across all sub-series, sorted by timestamp ascending.
func (r *JobMetricsResult) Total() []MetricPoint {
	if r == nil || len(r.Series) == 0 {
		return nil
	}
	if len(r.Series) == 1 {
		for _, pts := range r.Series {
			return pts
		}
	}
	if total, ok := r.Series["total"]; ok {
		return total
	}
	sums := make(map[string]float64)
	for _, pts := range r.Series {
		for _, p := range pts {
			sums[p.Timestamp] += p.Value
		}
	}
	total := make([]MetricPoint, 0, len(sums))
	for ts, v := range sums {
		total = append(total, MetricPoint{Timestamp: ts, Value: v})
	}
	sort.Slice(total, func(i, j int) bool { return total[i].Timestamp < total[j].Timestamp })
	return total
}

func isJSONNull(raw json.RawMessage) bool {
	return len(raw) == 0 || string(raw) == "null"
}

// TraceEntry represents a trace in a list.
type TraceEntry struct {
	ID            int                    `json:"id"`
	Time          string                 `json:"time"`
	TotalCallTime float64                `json:"total_call_time"`
	MemDelta      float64                `json:"mem_delta"` // memory increase in MB
	MetricName    string                 `json:"metric_name"`
	URI           string                 `json:"uri"`
	Context       map[string]interface{} `json:"context"`
}

// TracesResult wraps the traces list response.
type TracesResult struct {
	Traces []TraceEntry `json:"traces"`
}

// TraceSpan represents a span in a trace tree.
//
// The API populates the duration_ms and exclusive_duration_ms keys with
// values in seconds, not milliseconds. The JSON tags keep the wire names so
// --json stays a faithful pass-through; the Go fields carry the real unit.
type TraceSpan struct {
	ID                       string      `json:"id"`
	ParentID                 *string     `json:"parent_id"`
	Operation                string      `json:"operation"`
	Type                     string      `json:"type"`
	Description              *string     `json:"description"`
	DurationSeconds          float64     `json:"duration_ms"`
	ExclusiveDurationSeconds float64     `json:"exclusive_duration_ms"`
	Allocations              int64       `json:"allocations"`
	Children                 []TraceSpan `json:"children,omitempty"`
}

// TraceDetail contains full trace information including spans.
type TraceDetail struct {
	ID                int         `json:"id"`
	Time              string      `json:"time"`
	TotalCallTime     float64     `json:"total_call_time"`
	MemDelta          float64     `json:"mem_delta"` // memory increase in MB
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
	ID                int              `json:"id"`
	Name              string           `json:"name"`
	Message           string           `json:"message"`
	Status            string           `json:"status"`
	ErrorsCount       int              `json:"errors_count"`
	LastErrorAt       string           `json:"last_error_at"`
	RequestComponents json.RawMessage  `json:"request_components"`
	RequestURI        string           `json:"request_uri"`
	AppEnvironment    string           `json:"app_environment"`
	LatestError       *ErrorOccurrence `json:"latest_error,omitempty"`
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

// AnomalyEvent represents an anomaly event detected by Scout.
// Detail-only fields are populated by GetAnomalyEvent.
type AnomalyEvent struct {
	ID             int      `json:"id"`
	Metric         string   `json:"metric"`
	Endpoint       string   `json:"endpoint,omitempty"`
	Direction      string   `json:"direction"`
	Severity       string   `json:"severity"`
	StartedAt      string   `json:"started_at"`
	EndedAt        *string  `json:"ended_at,omitempty"`
	LastSeenAt     string   `json:"last_seen_at"`
	ZScore         float64  `json:"z_score"`
	CurrentValue   float64  `json:"current_value"`
	BaselineValue  float64  `json:"baseline_value"`
	Multiplier     *float64 `json:"multiplier,omitempty"`
	Open           bool     `json:"open"`
	Description    string   `json:"description"`
	SmartMonitorID *int     `json:"smart_monitor_id,omitempty"`
	DeployID       *int     `json:"deploy_id,omitempty"`

	// Detail-only fields
	BaselineStdDev  *float64             `json:"baseline_std_dev,omitempty"`
	DurationMinutes *int                 `json:"duration_minutes,omitempty"`
	SmartMonitor    *AnomalySmartMonitor `json:"smart_monitor,omitempty"`
	Deploy          *AnomalyDeploy       `json:"deploy,omitempty"`
}

// AnomalySmartMonitor is the joined smart monitor on an anomaly event detail.
type AnomalySmartMonitor struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Kind              string  `json:"kind"`
	SeverityThreshold float64 `json:"severity_threshold"`
	DurationMinutes   int     `json:"duration_minutes"`
}

// AnomalyDeploy is the joined deploy on an anomaly event detail.
type AnomalyDeploy struct {
	ID         int    `json:"id"`
	SHA        string `json:"sha"`
	DeployedAt string `json:"deployed_at"`
}

// AnomalyEventsResult wraps the anomaly events list response.
type AnomalyEventsResult struct {
	Events []AnomalyEvent `json:"anomaly_events"`
}

// AnomalyEventResult wraps a single anomaly event response.
type AnomalyEventResult struct {
	Event AnomalyEvent `json:"anomaly_event"`
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
	StartTime       string  `json:"start_time"`
	EndTime         string  `json:"end_time"`
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

// OrgUsage represents the response from the org usage endpoint.
type OrgUsage struct {
	BillingPeriod BillingPeriod `json:"billing_period"`
	PricingStyle  string        `json:"pricing_style"`
	APM           *APMUsage     `json:"apm,omitempty"`
	Nodes         *NodesUsage   `json:"nodes,omitempty"`
	Errors        *ErrorsUsage  `json:"errors,omitempty"`
	Logs          *LogsUsage    `json:"logs,omitempty"`
}

// BillingPeriod contains the start and end dates of the current billing cycle.
type BillingPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// APMUsage contains APM transaction usage for the billing period.
type APMUsage struct {
	TotalTransactions int64  `json:"total_transactions"`
	Limit             *int64 `json:"limit,omitempty"`
}

// NodesUsage contains billable node count (per-node pricing only).
type NodesUsage struct {
	ActiveCount int `json:"active_count"`
}

// ErrorsUsage contains error tracking usage for the billing period.
type ErrorsUsage struct {
	Count int64 `json:"count"`
	Limit int64 `json:"limit"`
}

// LogsUsage contains log ingestion usage for the billing period.
type LogsUsage struct {
	BytesUsed  int64  `json:"bytes_used"`
	LimitBytes *int64 `json:"limit_bytes,omitempty"`
}

// APIError represents an error from the Scout API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
}
