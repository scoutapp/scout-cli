package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatMultiplier(t *testing.T) {
	v15 := 1.5
	v32 := 3.2
	v10 := 10.0

	tests := []struct {
		name     string
		input    *float64
		expected string
	}{
		{"nil", nil, "—"},
		{"1.5x", &v15, "1.5x"},
		{"3.2x", &v32, "3.2x"},
		{"10.0x", &v10, "10.0x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatMultiplier(tt.input))
		})
	}
}
