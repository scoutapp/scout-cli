package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidMetricType(t *testing.T) {
	for _, valid := range validMetricTypes {
		assert.True(t, isValidMetricType(valid), valid)
	}
	assert.False(t, isValidMetricType("memory"))
	assert.False(t, isValidMetricType("Response_Time"))
	assert.False(t, isValidMetricType(""))
}

func TestUnitForMetricType(t *testing.T) {
	assert.Equal(t, "ms", unitForMetricType("response_time"))
	assert.Equal(t, "ms", unitForMetricType("queue_time"))
	assert.Equal(t, " rpm", unitForMetricType("throughput"))
	assert.Equal(t, "", unitForMetricType("apdex"))
}

func TestEndpointDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "base64 id decodes", input: "YXBpL21ldHJpY3Mvc2hvdw==", want: "api/metrics/show"},
		{name: "unpadded base64 id decodes", input: "YXBpL21ldHJpY3Mvc2hvdw", want: "api/metrics/show"},
		{name: "plain name passes through", input: "api/metrics/show", want: "api/metrics/show"},
		{name: "empty passes through", input: "", want: ""},
		{name: "non-printable decode passes through", input: "AAAA", want: "AAAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, endpointDisplayName(tt.input))
		})
	}
}
