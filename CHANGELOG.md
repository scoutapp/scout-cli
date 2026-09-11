# Changelog

## Pending

### Changed

- `scout usage --app <id>` (without `--by-day`) now filters to that one app instead of silently reporting every app, matching `--by-day --app <id>` (#29)
- `scout usage --billing-period --json` now wraps the per-app result in an object carrying `billing_period` and `server_total`, so a script can tell which window it got and what the billed figure was. Without `--billing-period` the shape is unchanged (#29)
- `scout jobs metrics --type latency --json` renames the API's `total` sub-series to `execution_time_total`, which is what it actually holds; other metric types are unchanged (#29)
- `scout errors show` omits the `Count:` line when the API reports 0, which it does for every error group even when `errors list` shows a real count (#29)
- `scout insights show` prints nested fields as indented `key: value` pairs and empty values as `—`, instead of leaking Go's `map[...]` and `<nil>` formatting (#29)

### Fixed

- `-n`/`--limit` is now applied to `--json` output, not just the human table, across every list command: `apps list`, `endpoints list`, `jobs list`, `traces list`, `errors list`, `errors occurrences`, `anomalies list`, `insights list`, `insights show`, and all four `usage` modes. Totals and percentages are still computed over the full result set (#29)
- `scout insights list` and `scout insights show` now honor `-n` at all; it previously had no effect on either (#29)
- `scout anomalies --endpoint` now accepts a Base64 endpoint id and a plain endpoint name as well as the scoped name from `anomalies list`, instead of silently matching nothing (#29)
- `--endpoint` and `--job` now reject a blank value client-side rather than passing it to the server (#29)
- A malformed `--job` value, or a job class name copy-pasted without its queue, now gets a clear client-side error instead of a raw `API error (500)` (#29)
- `scout auth logout` no longer fails on a machine with no config directory yet (#29)
- `--from`/`--to` now reject a reversed range and unreasonably large or negative relative durations, which previously produced silently empty or nonsensical results (#29)
- `--app 0` and negative app ids now report an invalid value instead of "no app specified" (#29)
- `scout traces show` on a 404 now explains that background job traces have no detail endpoint (#29)
- `scout insights show` field order is now stable; it previously ranged over a Go map and printed a different order every run (#29)

## Pending

### Added

- `scout billing` — show current billing period usage from the `/usage` API: billing period dates, pricing style, APM transactions (with plan limit), active nodes, errors, and log bytes (#8)
- `scout usage --billing-period` — scope usage to the exact billing period dates from the API and show the server-reported billed total alongside the per-app calculation (#8)
- `scout jobs list` and `scout jobs metrics` — list background job performance (throughput, execution time, latency, % of job time) and chart per-job metrics (`throughput`, `execution_time`, `latency`, `errors`, `allocations`) (#18)
- `scout traces list --job <job-id>` — list traces for a background job; `--job` also accepts the `queue/JobName` full name (#18)

### Changed

- `scout usage` labels its columns and totals as web transactions (`Web Transactions`, `% of Web`) — the throughput metric it is built on excludes background jobs (#8)
- Update Homebrew install docs for Homebrew 6 tap trust — document `brew trust --formula scoutapp/tap/scout-cli` for users upgrading from Homebrew 5 and a Brewfile (`brew bundle`) install form (#20)

### Fixed

- `scout traces list` and `scout traces show` no longer fail with `cannot unmarshal number ... into Go struct field ... mem_delta of type int64` — the API reports `mem_delta` as a float in megabytes, so it is now decoded as such and displayed as MB (#19)
- `scout traces list` Duration column and the legacy-trace header in `scout traces show` treated `total_call_time` as seconds; the API reports it in milliseconds, so an 83-second request no longer shows as `83483.9s` (#19)

## [0.4.0] - 2026-06-04

### Added

- `scout anomalies` and `scout anomalies show <id>` — list and inspect anomaly events (#15)
- `scout anomalies show` — include `severity_threshold` and `duration_minutes` in the smart monitor detail block (#16)

### Fixed

- API client now reports the HTTP status and response body for non-JSON error responses (e.g. a 404 from an undeployed endpoint) instead of a cryptic `invalid character 'N'` JSON parse error

## [0.3.3] - 2026-04-02

### Changed

- Install Homebrew binary as `scout` instead of `scout-cli`

## [0.3.2] - 2026-04-02

### Fixed

- Fix Homebrew formula generation — export `VERSION` and `REVISION` env vars for `envsubst`

## [0.3.1] - 2026-04-02

### Changed

- Switch Homebrew distribution from pre-built cask to source-based formula to avoid macOS Gatekeeper errors (#12)

## [0.3.0] - 2026-03-26

### Added

- `--toon` flag for TOON output format ([spec](https://github.com/toon-format/spec)) — a token-efficient, LLM-friendly alternative to JSON
- Auto-enable TOON (instead of JSON) when stdout is piped, for zero-config agent consumption
- `structuredOutput()` helper unifying `--json` and `--toon` output paths across all commands

## [0.2.0] - 2026-03-18

### Added

- `scout usage` command — show transaction usage across all apps for a given timeframe
  - Calculates total transactions per app from throughput time-series data
  - Fetches all apps in parallel for fast results
  - Automatically splits timeframes longer than 14 days into chunked API requests
  - `--by-day` flag for daily transaction totals
  - `--by-day --by-app` shows daily breakdown per app with % of day, % of total timeframe, and top endpoints
  - `--by-day --app <id>` shows daily breakdown for a single app with top endpoint
  - `--all` flag to include apps with zero usage
  - Displays timeframe and summary totals across all output modes
  - Supports `--from`, `--to`, `--json`, and `--limit` flags

## [0.1.0] - 2026-03-11

### Added

- Initial release of the Scout Monitoring CLI
- `scout auth login/logout/status` — API key authentication
- `scout apps list/show` — list and inspect applications
- `scout metrics get` — fetch metric data with ASCII charts (apdex, response_time, response_time_95th, errors, throughput, queue_time)
- `scout endpoints list/metrics` — list endpoints and view endpoint-level metrics
- `scout traces list/show` — list and inspect traces with span trees
- `scout errors list/show/occurrences` — list error groups and view occurrences
- `scout insights list/show` — view performance insights (n_plus_one, memory_bloat, slow_query)
- `scout setup` — show setup instructions for supported frameworks
- Global flags: `--json`, `--app`, `--from`, `--to`, `--no-color`, `--limit`
- Auto-detect piped output and switch to structured format
- Homebrew installation via `scoutapp/tap`
- CI/CD with GitHub Actions and GoReleaser
