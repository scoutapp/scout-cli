# Scout APM CLI

A command-line interface for [Scout Monitoring](https://scoutapm.com). Explore metrics, endpoints, traces, errors, and insights from the terminal.

Built in Go with [Cobra](https://github.com/spf13/cobra), [Lipgloss](https://github.com/charmbracelet/lipgloss), [BubbleTea](https://github.com/charmbracelet/bubbletea), and [asciigraph](https://github.com/guptarohit/asciigraph).

## Install

### Homebrew

```bash
brew install scoutapp/tap/scout-cli
```

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

Credentials are stored in `~/.config/scout-apm/config.json`. You can also set the `SCOUT_API_KEY` environment variable.

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

### Traces

```bash
scout traces list --endpoint YXBpL21ldHJpY3Mvc2hvdw== --app 6
scout traces show 12345 --app 6
```

### Errors

```bash
scout errors list --app 6
scout errors show 50560 --app 6
scout errors occurrences 50560 --app 6
```

### Usage

```bash
scout usage                                    # Transaction usage across all apps (last 3 hours)
scout usage --from 30d                         # Last 30 days
scout usage --from 7d --all                    # Include apps with zero usage
scout usage --by-day --from 30d                # Daily totals
scout usage --by-day --by-app --from 30d       # Daily breakdown per app with top endpoints
scout usage --by-day --app 6 --from 30d        # Daily breakdown for one app with top endpoint
scout usage --from 14d --json                  # JSON output
```

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
| `--json` | Output raw JSON |
| `--toon` | Output in [TOON](https://toon-format.org) format (auto-enabled when piped) |
| `--app <id>` | Application ID (or set `default_app_id` in config) |
| `--from <time>` | Start time — relative (`1h`, `7d`, `30m`, `2w`) or ISO 8601 |
| `--to <time>` | End time (default: now) |
| `-n, --limit <n>` | Max number of results to show |
| `--no-color` | Disable colors (also respects `NO_COLOR` env) |

## LLM / Agent Usage

When output is piped, Scout CLI automatically switches to [TOON](https://toon-format.org) format — a token-efficient structured format designed for LLM consumption. This means tools like Claude Code, scripts, and other agents get compact, parseable output by default.

```bash
# TOON output is automatic when piped
scout apps list | llm "which app has the most endpoints?"

# Force TOON in a terminal
scout metrics get --type response_time --app 6 --toon

# Use --json if you need raw JSON instead
scout metrics get --type response_time --app 6 --json
```

## Configuration

Config file: `~/.config/scout-apm/config.json`

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
