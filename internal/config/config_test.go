package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteConfig(t *testing.T) {
	// Use temp HOME to isolate test
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// XDG on macOS uses ~/Library/Application Support
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := Config{
		APIKey:       "test-key-123",
		APIURL:       "https://example.com",
		DefaultAppID: 42,
	}

	err := Write(cfg)
	require.NoError(t, err)

	got, err := Read()
	require.NoError(t, err)
	assert.Equal(t, cfg, got)
}

func TestReadMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, Config{}, cfg)
}

func TestClear(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := Write(Config{APIKey: "key"})
	require.NoError(t, err)

	err = Clear()
	require.NoError(t, err)

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "", cfg.APIKey)
}

func TestGetAPIKeyFromEnv(t *testing.T) {
	t.Setenv("SCOUT_API_KEY", "env-key")
	assert.Equal(t, "env-key", GetAPIKey())
}

func TestGetAPIURLDefault(t *testing.T) {
	os.Unsetenv("SCOUT_API_URL")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	assert.Equal(t, "https://scoutapm.com", GetAPIURL())
}

func TestGetAPIURLFromEnv(t *testing.T) {
	t.Setenv("SCOUT_API_URL", "https://custom.com")
	assert.Equal(t, "https://custom.com", GetAPIURL())
}
