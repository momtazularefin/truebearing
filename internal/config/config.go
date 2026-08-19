// Package config loads application configuration from environment variables.
package config

import "os"

// Config holds application-level configuration values.
type Config struct {
	DatabaseURL string
	RedisURL    string
	Port        string
	MetricsPort string
	LogLevel    string
}

// Load reads configuration from environment variables, falling back to
// sensible defaults for local development.
func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://truebearing:truebearing@localhost:5432/truebearing?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		Port:        getEnv("PORT", "8080"),
		MetricsPort: getEnv("METRICS_PORT", "9090"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv returns the value of the environment variable identified by key,
// or fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
