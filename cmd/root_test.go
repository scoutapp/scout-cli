package cmd

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withLimit sets -n for the duration of a test.
func withLimit(t *testing.T, n int) {
	t.Helper()
	prev := limitFlag
	limitFlag = n
	t.Cleanup(func() { limitFlag = prev })
}

// captureStdout runs fn with os.Stdout redirected and returns what it wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	prev := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = prev }()

	fn()
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestLimitSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	tests := []struct {
		name  string
		limit int
		shown []int
	}{
		{"no limit", 0, items},
		{"limit below total", 2, []int{1, 2}},
		{"limit equal to total", 5, items},
		{"limit above total", 99, items},
		{"negative limit ignored", -1, items},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withLimit(t, tt.limit)
			shown, total := limitSlice(items)
			assert.Equal(t, tt.shown, shown)
			assert.Equal(t, 5, total, "total is the count before limiting")
		})
	}
}

func TestLimitSliceEmpty(t *testing.T) {
	withLimit(t, 3)
	shown, total := limitSlice([]string{})
	assert.Empty(t, shown)
	assert.Equal(t, 0, total)
}

// -n used to apply only to the human table, leaving --json unlimited.
func TestStructuredOutputHonorsLimit(t *testing.T) {
	withLimit(t, 2)
	prevJSON := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = prevJSON })

	out := captureStdout(t, func() {
		shown, _ := limitSlice([]int{1, 2, 3, 4, 5})
		assert.True(t, structuredOutput(shown))
	})

	var decoded struct {
		Data []int `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	assert.Equal(t, []int{1, 2}, decoded.Data)
}

func TestValidateAppIDFlag(t *testing.T) {
	// Not passed: the default 0 means "use the configured default".
	assert.NoError(t, validateAppIDFlag(0, false))
	assert.NoError(t, validateAppIDFlag(6, true))

	err := validateAppIDFlag(0, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --app value: 0")

	err = validateAppIDFlag(-5, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --app value: -5")
}

func TestRequireFlagValue(t *testing.T) {
	got, err := requireFlagValue("endpoint", "  abc  ")
	require.NoError(t, err)
	assert.Equal(t, "abc", got)

	for _, blank := range []string{"", "   ", "\t\n"} {
		_, err := requireFlagValue("job", blank)
		require.Error(t, err)
		assert.Equal(t, "--job requires a value", err.Error())
	}
}
