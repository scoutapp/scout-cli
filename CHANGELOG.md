# Changelog

## Pending

### Added

- `Example:` blocks in `--help` for `scout jobs metrics`, `scout traces list`, `scout metrics get` and `scout endpoints metrics`, showing both the encoded-id and full-name forms (#29)
- Client-side `--type` validation on `scout metrics get` and `scout endpoints metrics`, listing the valid types instead of round-tripping to the server for a 422 (#29)
- `scout auth status` prints the config file path in use (#29)
- CI runs `gofmt -l` and `go vet ./...` (#29)

### Changed

- Table cells are capped at 60 display columns with an ellipsis, so one long URI no longer stretches a table past a normal terminal width; `--json` output is uncapped (#29)
- Chart titles show a decoded endpoint name and the app's name instead of a raw Base64 endpoint id and "App #6" (#29)

### Fixed

- Chart downsampling keeps the largest-magnitude value of each bucket instead of picking one point by index, so a spike between sampled points is no longer invisible in the plotted line (#29)
- Chart Min/Max/Avg and the plotted line exclude the trailing partial time bucket, which dragged the reported range toward zero (#29)
- Large millisecond chart values promote to seconds, so stats read `5.7s` rather than `5.7kms` (#29)
- The chart title renders once, as a styled header, rather than also as the chart library's own caption (#29)
- Charts always show `Summary:`, including a genuine zero, matching every other stat on the line (#29)
- README documents the credentials file path per platform; `os.UserConfigDir` resolves to `~/Library/Application Support` on macOS, not `~/.config` (#29)
- Root `--help` lists `anomalies`, `insights`, `usage` and `billing`, and `--from` help includes the `2w` unit (#29)
- README's errors and traces examples reference the list command that produces an id rather than hardcoded ids that no longer resolve (#29)
- `scout billing` usage bars show one block for any nonzero usage instead of rendering empty below 2.5% (#29)
- `scout apps show` reports `last_reported_at`, filling it from the list payload the single-app payload omits (#29)
- API errors no longer repeat an HTTP status text that the response body only duplicates ("Not Found: Not Found") (#29)
- String truncation in tables, the span tree and API error bodies counts graphemes rather than bytes, so it cannot split a multibyte character (#29)
- `scout setup <name>` matches framework names case-insensitively and its error points at `scout setup` with no arguments for the valid list (#29)

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
