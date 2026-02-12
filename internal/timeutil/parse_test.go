package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRelativeMinutes(t *testing.T) {
	before := time.Now().UTC()
	result, err := Parse("30m")
	after := time.Now().UTC()
	require.NoError(t, err)
	expected := before.Add(-30 * time.Minute)
	assert.WithinDuration(t, expected, result, after.Sub(before)+time.Second)
}

func TestParseRelativeHours(t *testing.T) {
	result, err := Parse("1h")
	require.NoError(t, err)
	expected := time.Now().UTC().Add(-1 * time.Hour)
	assert.WithinDuration(t, expected, result, 2*time.Second)
}

func TestParseRelativeDays(t *testing.T) {
	result, err := Parse("7d")
	require.NoError(t, err)
	expected := time.Now().UTC().AddDate(0, 0, -7)
	assert.WithinDuration(t, expected, result, 2*time.Second)
}

func TestParseRelativeWeeks(t *testing.T) {
	result, err := Parse("2w")
	require.NoError(t, err)
	expected := time.Now().UTC().AddDate(0, 0, -14)
	assert.WithinDuration(t, expected, result, 2*time.Second)
}

func TestParseISO8601(t *testing.T) {
	result, err := Parse("2026-01-15T10:00:00Z")
	require.NoError(t, err)
	expected := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse("foo")
	assert.Error(t, err)
}

func TestParseEmpty(t *testing.T) {
	_, err := Parse("")
	assert.Error(t, err)
}

func TestResolveTimeframeDefaults(t *testing.T) {
	from, to, err := ResolveTimeframe("", "")
	require.NoError(t, err)

	fromTime, _ := time.Parse(time.RFC3339, from)
	toTime, _ := time.Parse(time.RFC3339, to)

	// Default is last 3 hours
	diff := toTime.Sub(fromTime)
	assert.InDelta(t, 3*time.Hour, diff, float64(5*time.Second))
}

func TestResolveTimeframeWithFrom(t *testing.T) {
	from, to, err := ResolveTimeframe("1h", "")
	require.NoError(t, err)

	fromTime, _ := time.Parse(time.RFC3339, from)
	toTime, _ := time.Parse(time.RFC3339, to)

	diff := toTime.Sub(fromTime)
	assert.InDelta(t, 1*time.Hour, diff, float64(5*time.Second))
}
