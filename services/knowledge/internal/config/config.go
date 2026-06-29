package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr        = ":8083"
	DefaultServiceVersion  = "dev"
	DefaultEnvironment     = "local"
	DefaultMaxUploadBytes  = int64(32 << 20)
	DefaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	HTTPAddr        string
	ServiceVersion  string
	Environment     string
	DatabaseURL     string
	FileServiceURL  string
	RedisAddr       string
	ServiceToken    string
	MaxUploadBytes  int64
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        stringValue("KNOWLEDGE_HTTP_ADDR", DefaultHTTPAddr),
		ServiceVersion:  stringValue("KNOWLEDGE_SERVICE_VERSION", DefaultServiceVersion),
		Environment:     stringValue("KNOWLEDGE_ENV", DefaultEnvironment),
		DatabaseURL:     strings.TrimSpace(os.Getenv("DATABASE_URL")),
		FileServiceURL:  strings.TrimSpace(os.Getenv("FILE_SERVICE_BASE_URL")),
		RedisAddr:       strings.TrimSpace(os.Getenv("KNOWLEDGE_REDIS_ADDR")),
		ServiceToken:    strings.TrimSpace(os.Getenv("KNOWLEDGE_SERVICE_TOKEN")),
		MaxUploadBytes:  DefaultMaxUploadBytes,
		ShutdownTimeout: DefaultShutdownTimeout,
	}

	if raw := os.Getenv("KNOWLEDGE_MAX_UPLOAD_BYTES"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("KNOWLEDGE_MAX_UPLOAD_BYTES must be a positive integer")
		}
		cfg.MaxUploadBytes = value
	}

	if raw := os.Getenv("KNOWLEDGE_SHUTDOWN_TIMEOUT"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("KNOWLEDGE_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if err := validateHTTPURL("FILE_SERVICE_BASE_URL", cfg.FileServiceURL); err != nil {
		return Config{}, err
	}
	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("KNOWLEDGE_REDIS_ADDR is required")
	}

	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validateHTTPURL(name string, value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute http(s) URL", name)
	}
	if parsed.User != nil {
		return fmt.Errorf("%s must not contain credentials", name)
	}
	return nil
}
