# Scout APM CLI

A command-line interface for [Scout APM](https://scoutapm.com). View apps, metrics, endpoints, traces, errors, and insights from the terminal.

Built in Go with [Cobra](https://github.com/spf13/cobra), [Lipgloss](https://github.com/charmbracelet/lipgloss), [BubbleTea](https://github.com/charmbracelet/bubbletea), and [asciigraph](https://github.com/guptarohit/asciigraph).

## Install

```bash
go install github.com/scoutapm/scout-cli@latest
```

Or build from source:

```bash
git clone https://github.com/scoutapm/scout-cli.git
cd scout-cli
go build -o scout .
```

## Authentication

```bash
# Interactive login (prompts for API key)
scout auth login

# Non-interactive
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
| `--no-color` | Disable colors (also respects `NO_COLOR` env) |

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
