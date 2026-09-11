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

func TestLookupFramework(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantURL string
		found   bool
	}{
		{input: "rails", want: "rails", wantURL: "https://scoutapm.com/docs/ruby", found: true},
		{input: "Rails", want: "rails", wantURL: "https://scoutapm.com/docs/ruby", found: true},
		{input: "RAILS", want: "rails", wantURL: "https://scoutapm.com/docs/ruby", found: true},
		{input: "FastAPI", want: "fastapi", wantURL: "https://scoutapm.com/docs/python/fastapi", found: true},
		{input: "django", want: "django", wantURL: "https://scoutapm.com/docs/python/django", found: true},
		{input: "sinatra", want: "sinatra", wantURL: "https://scoutapm.com/docs/ruby/sinatra", found: true},
		{input: "express", want: "express", wantURL: "https://scoutapm.com/docs/node/express", found: true},
		{input: "sidekiq", want: "sidekiq", wantURL: "https://scoutapm.com/docs/ruby#instrumented-libraries", found: true},
		{input: "ruby", found: false},
		{input: "cobol", found: false},
		{input: "", found: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, docsURL, ok := lookupFramework(tt.input)
			assert.Equal(t, tt.found, ok)
			if tt.found {
				assert.Equal(t, tt.want, name)
				assert.Equal(t, tt.wantURL, docsURL)
			} else {
				assert.Empty(t, name)
				assert.Empty(t, docsURL)
			}
		})
	}
}
