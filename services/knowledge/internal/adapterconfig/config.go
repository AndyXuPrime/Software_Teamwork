package adapterconfig

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr         = ":8083"
	DefaultServiceVersion   = "dev"
	DefaultEnvironment      = "local"
	DefaultVendorRuntimeURL = "http://127.0.0.1:9380"
	DefaultShutdownTimeout  = 10 * time.Second
)

type Config struct {
	HTTPAddr         string
	ServiceVersion   string
	Environment      string
	VendorRuntimeURL string
	DatabaseURL      string
	ShutdownTimeout  time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:       stringValue("KNOWLEDGE_HTTP_ADDR", DefaultHTTPAddr),
		ServiceVersion: stringValue("KNOWLEDGE_SERVICE_VERSION", DefaultServiceVersion),
		Environment:    stringValue("KNOWLEDGE_ENV", DefaultEnvironment),
		ShutdownTimeout: DefaultShutdownTimeout,
	}
	cfg.VendorRuntimeURL = trimTrailingSlash(stringValue("VENDOR_RUNTIME_URL", DefaultVendorRuntimeURL))
	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))

	if raw := os.Getenv("KNOWLEDGE_SHUTDOWN_TIMEOUT"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, fmt.Errorf("KNOWLEDGE_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}

	if strings.TrimSpace(cfg.VendorRuntimeURL) == "" {
		return Config{}, fmt.Errorf("VENDOR_RUNTIME_URL is required")
	}

	return cfg, nil
}

func stringValue(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func trimTrailingSlash(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
