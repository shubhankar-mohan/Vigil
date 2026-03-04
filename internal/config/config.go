package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PrometheusURL      string        `yaml:"prometheus_url"`
	PrometheusUser     string        `yaml:"prometheus_user"`
	PrometheusPassword string        `yaml:"prometheus_password"`
	LokiURL            string        `yaml:"loki_url"`
	LokiUser           string        `yaml:"loki_user"`
	LokiPassword       string        `yaml:"loki_password"`
	GrafanaURL         string        `yaml:"grafana_url"`
	GrafanaAPIToken    string        `yaml:"grafana_api_token"`
	EvalIntervalStr    string        `yaml:"eval_interval"`
	ListenAddr         string        `yaml:"listen_addr"`
	DBPath             string        `yaml:"db_path"`

	EvalInterval time.Duration `yaml:"-"`
}

// Load reads config from a YAML file, then applies env var overrides.
// File search order: CONFIG_FILE env var → ./vigil.yml → /etc/vigil/vigil.yml
func Load() *Config {
	cfg := &Config{
		PrometheusURL: "http://prometheus:9090",
		LokiURL:       "http://loki:3100",
		EvalInterval:  30 * time.Second,
		ListenAddr:    ":8080",
		DBPath:        "/data/vigil.db",
	}

	// Load from YAML file
	configFile := findConfigFile()
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("failed to read config file %s: %v", configFile, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Fatalf("failed to parse config file %s: %v", configFile, err)
		}
		log.Printf("loaded config from %s", configFile)
	}

	// Parse eval_interval string from YAML
	if cfg.EvalIntervalStr != "" {
		if d, err := time.ParseDuration(cfg.EvalIntervalStr); err == nil {
			cfg.EvalInterval = d
		}
	}

	// Env vars override YAML values
	envOverride(&cfg.PrometheusURL, "PROMETHEUS_URL")
	envOverride(&cfg.PrometheusUser, "PROMETHEUS_USER")
	envOverride(&cfg.PrometheusPassword, "PROMETHEUS_PASSWORD")
	envOverride(&cfg.LokiURL, "LOKI_URL")
	envOverride(&cfg.LokiUser, "LOKI_USER")
	envOverride(&cfg.LokiPassword, "LOKI_PASSWORD")
	envOverride(&cfg.GrafanaURL, "GRAFANA_URL")
	envOverride(&cfg.GrafanaAPIToken, "GRAFANA_API_TOKEN")
	envOverride(&cfg.ListenAddr, "LISTEN_ADDR")
	envOverride(&cfg.DBPath, "DB_PATH")

	if v := os.Getenv("EVAL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.EvalInterval = d
		}
	}

	return cfg
}

func findConfigFile() string {
	// Explicit path via env var
	if f := os.Getenv("CONFIG_FILE"); f != "" {
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}

	// Search common locations
	for _, path := range []string{"vigil.yml", "/etc/vigil/vigil.yml"} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func envOverride(target *string, key string) {
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}
