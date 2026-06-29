package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr        = ":8080"
	DefaultServiceVersion  = "0.1.0"
	DefaultEnvironment     = "local"
	DefaultMaxBodyBytes    = int64(10 << 20)
	DefaultRequestTimeout  = 30 * time.Second
	DefaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	HTTPAddr             string
	ServiceVersion       string
	Environment          string
	MaxBodyBytes         int64
	RequestTimeout       time.Duration
	ShutdownTimeout      time.Duration
	CORSAllowedOrigins   []string
	CORSAllowedMethods   []string
	CORSAllowedHeaders   []string
	CORSAllowCredentials bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           stringValue("GATEWAY_HTTP_ADDR", DefaultHTTPAddr),
		ServiceVersion:     stringValue("GATEWAY_SERVICE_VERSION", DefaultServiceVersion),
		Environment:        stringValue("GATEWAY_ENV", DefaultEnvironment),
		MaxBodyBytes:       DefaultMaxBodyBytes,
		RequestTimeout:     DefaultRequestTimeout,
		ShutdownTimeout:    DefaultShutdownTimeout,
		CORSAllowedOrigins: csvValue("GATEWAY_CORS_ALLOWED_ORIGINS", []string{"*"}),
		CORSAllowedMethods: csvValue("GATEWAY_CORS_ALLOWED_METHODS", []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders: csvValue("GATEWAY_CORS_ALLOWED_HEADERS", []string{"Authorization", "Content-Type", "X-Request-Id"}),
	}

	if raw := os.Getenv("GATEWAY_MAX_BODY_BYTES"); raw != "" {
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("GATEWAY_MAX_BODY_BYTES must be a positive integer")
		}
		cfg.MaxBodyBytes = value
	}

	if raw := os.Getenv("GATEWAY_REQUEST_TIMEOUT"); raw != "" {
		value, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("GATEWAY_REQUEST_TIMEOUT must be a positive duration")
		}
		cfg.RequestTimeout = value
	}

	if raw := os.Getenv("GATEWAY_SHUTDOWN_TIMEOUT"); raw != "" {
		value, err := time.ParseDuration(strings.TrimSpace(raw))
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("GATEWAY_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}

	if raw := os.Getenv("GATEWAY_CORS_ALLOW_CREDENTIALS"); raw != "" {
		value, err := strconv.ParseBool(strings.TrimSpace(raw))
		if err != nil {
			return Config{}, fmt.Errorf("GATEWAY_CORS_ALLOW_CREDENTIALS must be a boolean")
		}
		cfg.CORSAllowCredentials = value
	}

	if strings.TrimSpace(cfg.HTTPAddr) == "" {
		return Config{}, fmt.Errorf("GATEWAY_HTTP_ADDR must not be empty")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return Config{}, fmt.Errorf("GATEWAY_CORS_ALLOWED_ORIGINS must not be empty")
	}

	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func csvValue(key string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
