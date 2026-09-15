package cmd

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameworkTableIsWellFormed(t *testing.T) {
	seen := make(map[string]bool, len(frameworks))

	for _, f := range frameworks {
		t.Run(f.name, func(t *testing.T) {
			assert.NotEmpty(t, f.name)
			assert.NotEmpty(t, f.desc)
			assert.False(t, seen[f.name], "duplicate framework entry")
			seen[f.name] = true

			u, err := url.Parse(f.docsURL)
			require.NoError(t, err)
			assert.Equal(t, "https", u.Scheme)
			assert.Equal(t, "scoutapm.com", u.Host)
			assert.True(t, strings.HasPrefix(u.Path, "/docs/"), "docs URL should live under /docs/: %s", f.docsURL)

			// Scout's docs are organized by language, so /docs/<framework> 404s.
			assert.NotEqual(t, "/docs/"+f.name, u.Path, "docs URL must not be derived from the framework name")
		})
	}
}

func TestFrameworkDocsURLLookup(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"rails", "https://scoutapm.com/docs/ruby"},
		{"django", "https://scoutapm.com/docs/python/django"},
		{"sinatra", "https://scoutapm.com/docs/ruby/sinatra"},
		{"express", "https://scoutapm.com/docs/node/express"},
		{"sidekiq", "https://scoutapm.com/docs/ruby#instrumented-libraries"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := frameworkDocsURL(tt.name)
			require.True(t, found)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFrameworkDocsURLUnknown(t *testing.T) {
	got, found := frameworkDocsURL("cobol")
	assert.False(t, found)
	assert.Empty(t, got)
}
