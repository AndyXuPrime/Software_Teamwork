package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	DefaultHTTPAddr        = ":8082"
	DefaultMaxUploadBytes  = int64(32 << 20)
	DefaultStorageBackend  = "memory"
	DefaultShutdownTimeout = 10 * time.Second
)

type Config struct {
	HTTPAddr        string
	MaxUploadBytes  int64
	StorageBackend  string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        stringValue("FILE_HTTP_ADDR", DefaultHTTPAddr),
		StorageBackend:  stringValue("FILE_STORAGE_BACKEND", DefaultStorageBackend),
		MaxUploadBytes:  DefaultMaxUploadBytes,
		ShutdownTimeout: DefaultShutdownTimeout,
	}

	if raw := os.Getenv("FILE_MAX_UPLOAD_BYTES"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("FILE_MAX_UPLOAD_BYTES must be a positive integer")
		}
		cfg.MaxUploadBytes = value
	}

	if raw := os.Getenv("FILE_SHUTDOWN_TIMEOUT"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("FILE_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}

	if cfg.StorageBackend != "memory" {
		return Config{}, fmt.Errorf("FILE_STORAGE_BACKEND=%q is not implemented; supported value: memory", cfg.StorageBackend)
	}

	return cfg, nil
}

func stringValue(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
