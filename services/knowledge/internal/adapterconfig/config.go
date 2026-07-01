package adapterconfig

import (
	"fmt"
	"os"
	"strconv"
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
	HTTPAddr           string
	MCPAddr            string
	ServiceVersion     string
	Environment        string
	VendorRuntimeURL   string
	VendorRerankID     string
	DatabaseURL        string
	AutoStartIngestion bool
	ShutdownTimeout    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        stringValue("KNOWLEDGE_HTTP_ADDR", DefaultHTTPAddr),
		MCPAddr:         strings.TrimSpace(os.Getenv("KNOWLEDGE_MCP_ADDR")),
		ServiceVersion:  stringValue("KNOWLEDGE_SERVICE_VERSION", DefaultServiceVersion),
		Environment:     stringValue("KNOWLEDGE_ENV", DefaultEnvironment),
		ShutdownTimeout: DefaultShutdownTimeout,
	}
	cfg.VendorRuntimeURL = trimTrailingSlash(stringValue("VENDOR_RUNTIME_URL", DefaultVendorRuntimeURL))
	cfg.VendorRerankID = strings.TrimSpace(os.Getenv("KNOWLEDGE_VENDOR_RERANK_ID"))
	cfg.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	cfg.AutoStartIngestion = boolValue("KNOWLEDGE_AUTO_START_INGESTION", true)

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

func boolValue(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
