package cmd

import (
	"errors"
	"testing"

	"github.com/scoutapm/scout/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceShowError(t *testing.T) {
	// A 404 is most often a job trace id, which has no detail endpoint.
	err := traceShowError(&api.APIError{StatusCode: 404, Message: "Record Not Found"})
	var apiErr *api.APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, 404, apiErr.StatusCode)
	assert.Contains(t, apiErr.Message, "Record Not Found")
	assert.Contains(t, apiErr.Message, "background job traces")

	// Other statuses pass through untouched.
	original := &api.APIError{StatusCode: 500, Message: "Internal Server Error"}
	assert.Same(t, original, traceShowError(original))

	plain := errors.New("connection refused")
	assert.Same(t, plain, traceShowError(plain))
}
