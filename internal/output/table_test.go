package output

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRenderTableCapsCellWidth(t *testing.T) {
	long := strings.Repeat("a", 200)

	out := RenderTable([]string{"URI"}, [][]string{{long}})

	assert.NotContains(t, out, strings.Repeat("a", maxCellWidth+1))
	assert.Contains(t, out, "…")
}

func TestRenderTableKeepsShortCells(t *testing.T) {
	out := RenderTable([]string{"Name"}, [][]string{{"UsersController#show"}})

	assert.Contains(t, out, "UsersController#show")
	assert.NotContains(t, out, "…")
}

func TestTruncateIsRuneAware(t *testing.T) {
	// Byte slicing would cut these three-byte runes mid-character.
	got := Truncate("日本語です", 5)

	assert.True(t, utf8.ValidString(got))
	assert.LessOrEqual(t, lipgloss.Width(got), 5)
	assert.True(t, strings.HasSuffix(got, "…"))
	assert.Equal(t, "日本語です", Truncate("日本語です", 20))
}

func TestTruncatePreservesANSI(t *testing.T) {
	got := Truncate("\x1b[1mhello world\x1b[0m", 6)

	assert.Contains(t, got, "\x1b[1m")
	assert.LessOrEqual(t, lipgloss.Width(got), 6)
	assert.Contains(t, got, "hello")
}
