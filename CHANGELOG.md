# Changelog

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
- Auto-detect piped output and switch to JSON
- Homebrew installation via `scoutapp/tap`
- CI/CD with GitHub Actions and GoReleaser
