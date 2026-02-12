package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey       string `json:"api_key,omitempty"`
	APIURL       string `json:"api_url,omitempty"`
	DefaultAppID int    `json:"default_app_id,omitempty"`
}

const (
	configDir  = "scout-apm"
	configFile = "config.json"
)

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configDir, configFile), nil
}

func Read() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Write(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Clear() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte("{}"), 0600)
}

func GetAPIKey() string {
	if key := os.Getenv("SCOUT_API_KEY"); key != "" {
		return key
	}
	cfg, err := Read()
	if err != nil {
		return ""
	}
	return cfg.APIKey
}

func GetAPIURL() string {
	if url := os.Getenv("SCOUT_API_URL"); url != "" {
		return url
	}
	cfg, err := Read()
	if err != nil || cfg.APIURL == "" {
		return "https://scoutapm.com"
	}
	return cfg.APIURL
}

func Path() string {
	p, _ := configPath()
	return p
}
