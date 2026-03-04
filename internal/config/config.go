package config

import (
	"os"
	"time"
)

type Config struct {
	PrometheusURL      string
	PrometheusUser     string
	PrometheusPassword string
	LokiURL            string
	LokiUser           string
	LokiPassword       string
	GrafanaURL         string
	GrafanaAPIToken    string
	EvalInterval       time.Duration
	ListenAddr         string
	DBPath             string
}

func Load() *Config {
	return &Config{
		PrometheusURL:      envOrDefault("PROMETHEUS_URL", "http://prometheus:9090"),
		PrometheusUser:     envOrDefault("PROMETHEUS_USER", ""),
		PrometheusPassword: envOrDefault("PROMETHEUS_PASSWORD", ""),
		LokiURL:            envOrDefault("LOKI_URL", "http://loki:3100"),
		LokiUser:           envOrDefault("LOKI_USER", ""),
		LokiPassword:       envOrDefault("LOKI_PASSWORD", ""),
		GrafanaURL:         envOrDefault("GRAFANA_URL", ""),
		GrafanaAPIToken:    envOrDefault("GRAFANA_API_TOKEN", ""),
		EvalInterval:       parseDuration("EVAL_INTERVAL", 30*time.Second),
		ListenAddr:         envOrDefault("LISTEN_ADDR", ":8080"),
		DBPath:             envOrDefault("DB_PATH", "/data/vigil.db"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
