# Scout APM CLI

A command-line interface for [Scout Monitoring](https://scoutapm.com). Explore metrics, endpoints, background jobs, traces, errors, insights, and billing usage from the terminal.

Built in Go with [Cobra](https://github.com/spf13/cobra), [Lipgloss](https://github.com/charmbracelet/lipgloss), [BubbleTea](https://github.com/charmbracelet/bubbletea), and [asciigraph](https://github.com/guptarohit/asciigraph).

## Install

### Homebrew

```bash
brew install scoutapp/tap/scout-cli
```

On Homebrew 6 and later, third-party taps require [tap trust](https://docs.brew.sh/Tap-Trust). The fully qualified name above trusts only the `scout-cli` formula, not the whole tap.

**Upgrading from Homebrew 5?** If `brew upgrade` warns that `scoutapp/tap` is not trusted, grant trust and upgrade again:

```bash
brew trust --formula scoutapp/tap/scout-cli   # or: brew trust scoutapp/tap
brew upgrade scout-cli
```

**Brewfile / `brew bundle`:**

```bash
brew bundle install --file=- <<'BREWFILE'
tap "scoutapp/tap"
brew "scoutapp/tap/scout-cli", trusted: true
BREWFILE
```

Use `tap "scoutapp/tap", trusted: true` instead if you want to trust the whole tap.

### Download binary

Pre-built binaries for macOS, Linux, and Windows are available on the [GitHub Releases](https://github.com/scoutapp/scout-cli/releases) page.

### Build from source

```bash
git clone https://github.com/scoutapp/scout-cli.git
cd scout-cli
go build -o scout .
```

## Authentication

```bash
# Login
scout auth login --key YOUR_API_KEY

# Check auth status
scout auth status

# Clear credentials
scout auth logout
```

Credentials are stored in a per-user config file whose location follows the platform convention:

| Platform | Path |
| --- | --- |
| macOS | `~/Library/Application Support/scout-apm/config.json` |
| Linux | `$XDG_CONFIG_HOME/scout-apm/config.json`, defaulting to `~/.config/scout-apm/config.json` |
| Windows | `%AppData%\scout-apm\config.json` |

`scout auth status` prints the path in use. You can also set the `SCOUT_API_KEY` environment variable, which takes precedence over the config file.

## Usage

### Apps

```bash
scout apps list
scout apps show 6
```

### Metrics

```bash
scout metrics get --type response_time --app 6
scout metrics get --type throughput --app 6 --from 7d
```

Valid metric types: `apdex`, `response_time`, `response_time_95th`, `errors`, `throughput`, `queue_time`

### Endpoints

```bash
scout endpoints list --app 6
scout endpoints metrics --endpoint YXBpL21ldHJpY3Mvc2hvdw== --type response_time --app 6
```

### Background Jobs

```bash
scout jobs list --app 6
scout jobs metrics --job ZGVmYXVsdC9NeVdvcmtlcg== --type execution_time --app 6
scout jobs metrics --job default/MyWorker --type throughput --app 6   # queue/JobName also works
scout traces list --job ZGVmYXVsdC9NeVdvcmtlcg== --app 6
```

Valid job metric types: `throughput`, `execution_time`, `latency`, `errors`, `allocations`

Job IDs are in the `scout jobs list --json` output (`job_id`), or pass the job's `queue/JobName` full name and the CLI encodes it for you. Job traces can be listed but not yet shown in detail (`scout traces show` supports endpoint traces only).

### Traces

```bash
scout traces list --endpoint YXBpL21ldHJpY3Mvc2hvdw== --app 6
scout traces list --job ZGVmYXVsdC9NeVdvcmtlcg== --app 6
scout traces show <trace-id> --app 6
```

Trace ids come from `scout traces list`.

### Errors

```bash
scout errors list --app 6
scout errors show <error-group-id> --app 6
scout errors occurrences <error-group-id> --app 6
```

Error group ids come from `scout errors list`.

### Anomalies

```bash
scout anomalies --app 6                                # List anomaly events (default: all states)
scout anomalies --app 6 --state open                   # Only open anomalies
scout anomalies --app 6 --metric response_time         # Filter by metric
scout anomalies --app 6 --endpoint YXBpL21ldHJpY3M=    # Filter by endpoint
scout anomalies show 1234 --app 6                      # Detail with smart_monitor and deploy context
```

### Usage

```bash
scout usage                                    # Web transaction usage across all apps (last 3 hours)
scout usage --from 30d                         # Last 30 days
scout usage --from 7d --all                    # Include apps with zero usage
scout usage --by-day --from 30d                # Daily totals
scout usage --by-day --by-app --from 30d       # Daily breakdown per app with top endpoints
scout usage --by-day --app 6 --from 30d        # Daily breakdown for one app with top endpoint
scout usage --from 14d --json                  # JSON output
scout usage --billing-period                   # Current billing period, with the server-side billed total
scout usage --by-day --by-app --billing-period # Daily per-app breakdown over the billing period
```

`scout usage` counts web transactions from the throughput metric; background jobs are excluded. The billed total, which includes jobs, is shown by `scout billing` and by `scout usage --billing-period`.

### Billing

```bash
scout billing          # Current billing period: APM transactions, nodes, errors, logs
scout billing --json   # Raw JSON from the /usage API endpoint
```

Shows the exact usage Scout bills against for the current billing period, including plan limits where they apply. `scout usage` estimates web transactions from throughput metrics; `scout billing` reports the billed figures, which also include background jobs.

### Insights

```bash
scout insights list --app 6
scout insights show --type slow_query --app 6
```

Valid insight types: `n_plus_one`, `memory_bloat`, `slow_query`

### Setup

```bash
scout setup           # List supported frameworks
scout setup rails     # Show setup docs for a framework
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON (auto-enabled when piped) |
| `--app <id>` | Application ID (or set `default_app_id` in config) |
| `--from <time>` | Start time — relative (`1h`, `7d`, `30m`, `2w`) or ISO 8601 |
| `--to <time>` | End time (default: now) |
| `-n, --limit <n>` | Max number of results to show |
| `--no-color` | Disable colors (also respects `NO_COLOR` env) |

## LLM / Agent Usage

When output is piped, Scout CLI automatically switches to JSON. Tools like Claude Code, scripts, and other agents get structured, parseable output by default — no flags needed. Human-readable tables and charts are used only when writing to a terminal.

```bash
# JSON output is automatic when piped
scout apps list | llm "which app has the most endpoints?"

# Force JSON in a terminal
scout metrics get --type response_time --app 6 --json
```

## Configuration

Config file location (see [Authentication](#authentication) for the full table): `~/Library/Application Support/scout-apm/config.json` on macOS, `~/.config/scout-apm/config.json` on Linux. Run `scout auth status` to print the path in use.

```json
{
  "api_key": "your-api-key",
  "api_url": "https://scoutapm.com",
  "default_app_id": 6
}
```

Environment variable overrides: `SCOUT_API_KEY`, `SCOUT_API_URL`.

## Testing

```bash
go test ./...
```

## Shell Completion

```bash
scout completion bash   # Bash
scout completion zsh    # Zsh
scout completion fish   # Fish
```

## Links

- [Scout Monitoring](https://scoutapm.com) — Application performance monitoring
- [Documentation](https://scoutapm.com/docs)
- [GitHub Releases](https://github.com/scoutapp/scout-cli/releases)
