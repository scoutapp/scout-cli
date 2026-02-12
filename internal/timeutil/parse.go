package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Parse(input string) (time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return time.Time{}, fmt.Errorf("empty time input")
	}

	// Try relative time: 30m, 1h, 7d, 2w
	if len(input) >= 2 {
		unit := input[len(input)-1]
		numStr := input[:len(input)-1]
		if n, err := strconv.Atoi(numStr); err == nil {
			now := time.Now().UTC()
			switch unit {
			case 'm':
				return now.Add(-time.Duration(n) * time.Minute), nil
			case 'h':
				return now.Add(-time.Duration(n) * time.Hour), nil
			case 'd':
				return now.AddDate(0, 0, -n), nil
			case 'w':
				return now.AddDate(0, 0, -n*7), nil
			}
		}
	}

	// Try ISO 8601
	t, err := time.Parse(time.RFC3339, input)
	if err == nil {
		return t, nil
	}

	// Try without timezone
	t, err = time.Parse("2006-01-02T15:04:05", input)
	if err == nil {
		return t, nil
	}

	// Try date only
	t, err = time.Parse("2006-01-02", input)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %q (use relative like 1h, 7d, 30m, 2w or ISO 8601)", input)
}

// ResolveTimeframe returns (from, to) as ISO 8601 strings.
// Defaults to last 3 hours if neither is provided.
func ResolveTimeframe(fromStr, toStr string) (string, string, error) {
	now := time.Now().UTC()

	var to time.Time
	if toStr == "" {
		to = now
	} else {
		var err error
		to, err = Parse(toStr)
		if err != nil {
			return "", "", fmt.Errorf("invalid --to: %w", err)
		}
	}

	var from time.Time
	if fromStr == "" {
		from = now.Add(-3 * time.Hour)
	} else {
		var err error
		from, err = Parse(fromStr)
		if err != nil {
			return "", "", fmt.Errorf("invalid --from: %w", err)
		}
	}

	return from.Format(time.RFC3339), to.Format(time.RFC3339), nil
}
