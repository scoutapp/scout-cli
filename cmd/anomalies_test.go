package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAnomalyEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"scoped name passes through", "Controller/users/index", "Controller/users/index"},
		{"job scope passes through", "Job/default/MyWorker", "Job/default/MyWorker"},
		{"plain name gains the Controller scope", "users/index", "Controller/users/index"},
		{"base64 endpoint id is decoded and scoped", "dXNlcnMvaW5kZXg=", "Controller/users/index"},
		{"unpadded base64 id", "dXNlcnMvaW5kZXg", "Controller/users/index"},
		{"surrounding whitespace is trimmed", "  Controller/users/index  ", "Controller/users/index"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveAnomalyEndpoint(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolveAnomalyEndpointRejectsBadValues(t *testing.T) {
	_, err := resolveAnomalyEndpoint("")
	require.Error(t, err)
	assert.Equal(t, "--endpoint requires a value", err.Error())

	// No slash and not decodable: nothing sensible to send to the API.
	_, err = resolveAnomalyEndpoint("not base64!!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --endpoint")
}

func TestHasMetricScope(t *testing.T) {
	assert.True(t, hasMetricScope("Controller/users/index"))
	assert.True(t, hasMetricScope("Job/default/MyWorker"))
	assert.False(t, hasMetricScope("users/index"))
	assert.False(t, hasMetricScope("users"))
	assert.False(t, hasMetricScope("/leading-slash"))
	assert.False(t, hasMetricScope("Controller/"))
}

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
