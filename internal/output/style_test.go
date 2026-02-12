package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMs(t *testing.T) {
	assert.Equal(t, "245ms", FormatMs(245))
	assert.Equal(t, "1.2s", FormatMs(1200))
	assert.Equal(t, "0ms", FormatMs(0))
}

func TestFormatSeconds(t *testing.T) {
	assert.Equal(t, "1.2s", FormatSeconds(1.245))
	assert.Equal(t, "245ms", FormatSeconds(0.245))
}

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "0", FormatNumber(0))
	assert.Equal(t, "999", FormatNumber(999))
	assert.Equal(t, "1,000", FormatNumber(1000))
	assert.Equal(t, "1,247,000", FormatNumber(1247000))
}

func TestFormatRPM(t *testing.T) {
	assert.Equal(t, "245 rpm", FormatRPM(245))
	assert.Equal(t, "1.2k rpm", FormatRPM(1200))
}

func TestFormatPercent(t *testing.T) {
	assert.Equal(t, "0.0%", FormatPercent(0))
	assert.Equal(t, "50.0%", FormatPercent(0.5))
	assert.Equal(t, "0.3%", FormatPercent(0.003))
}

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "500 B", FormatBytes(500))
	assert.Equal(t, "2.0 KB", FormatBytes(2000))
	assert.Equal(t, "11.8 MB", FormatBytes(11800000))
	assert.Equal(t, "1.5 GB", FormatBytes(1500000000))
}
