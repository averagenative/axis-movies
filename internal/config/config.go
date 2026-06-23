// Package config loads runtime configuration from an optional YAML file,
// overlaid by AXIS_* environment variables. Env always wins over file.
package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the fully resolved runtime configuration.
type Config struct {
	// HTTPAddr is the listen address for the API/UI server.
	HTTPAddr string `yaml:"http_addr"`
	// APIKey authenticates v3 API requests (X-Api-Key header or apikey query).
	APIKey string `yaml:"api_key"`
	// DatabaseURL is the Postgres connection string (postgres://...).
	DatabaseURL string `yaml:"database_url"`
	// LogLevel is one of debug|info|warn|error.
	LogLevel string `yaml:"log_level"`
	// LogFormat is text|json.
	LogFormat string `yaml:"log_format"`
	// CompatAppName is the value reported as appName by /system/status.
	// Defaults to "Radarr" so ecosystem tools (Prowlarr, Overseerr, mobile
	// clients) that string-match the app type connect cleanly.
	CompatAppName string `yaml:"compat_app_name"`
	// InstanceName is the human-facing instance label.
	InstanceName string `yaml:"instance_name"`
}

func defaults() Config {
	return Config{
		HTTPAddr:      ":7878", // Radarr's default port, for drop-in parity.
		DatabaseURL:   "postgres://axis:axis@localhost:5432/axis_movies?sslmode=disable",
		LogLevel:      "info",
		LogFormat:     "text",
		CompatAppName: "Radarr",
		InstanceName:  "Axis Movies",
	}
}

// Load resolves configuration from the file at AXIS_CONFIG (if set/present)
// then applies AXIS_* environment overrides.
func Load() (Config, error) {
	cfg := defaults()

	if path := os.Getenv("AXIS_CONFIG"); path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	applyEnv(&cfg)

	if cfg.APIKey == "" {
		key, err := randomKey()
		if err != nil {
			return Config{}, err
		}
		cfg.APIKey = key
		fmt.Fprintf(os.Stderr, "axis-movies: no api key configured, generated ephemeral key: %s\n", key)
	}

	return cfg, nil
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %q: %w", path, err)
	}
	return nil
}

func applyEnv(cfg *Config) {
	setString(&cfg.HTTPAddr, "AXIS_HTTP_ADDR")
	setString(&cfg.APIKey, "AXIS_API_KEY")
	setString(&cfg.DatabaseURL, "AXIS_DATABASE_URL")
	setString(&cfg.LogLevel, "AXIS_LOG_LEVEL")
	setString(&cfg.LogFormat, "AXIS_LOG_FORMAT")
	setString(&cfg.CompatAppName, "AXIS_COMPAT_APP_NAME")
	setString(&cfg.InstanceName, "AXIS_INSTANCE_NAME")
}

func setString(dst *string, env string) {
	if v, ok := os.LookupEnv(env); ok {
		*dst = v
	}
}

func randomKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
