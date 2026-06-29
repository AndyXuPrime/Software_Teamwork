package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadValidatesUploadDependencies(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable")
	t.Setenv("FILE_SERVICE_BASE_URL", "http://localhost:8082")
	t.Setenv("KNOWLEDGE_REDIS_ADDR", "localhost:6379")
	t.Setenv("KNOWLEDGE_MAX_UPLOAD_BYTES", "1024")
	t.Setenv("KNOWLEDGE_SHUTDOWN_TIMEOUT", "7s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.FileServiceURL != "http://localhost:8082" || cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("dependency config = %+v", cfg)
	}
	if cfg.MaxUploadBytes != 1024 {
		t.Fatalf("MaxUploadBytes = %d", cfg.MaxUploadBytes)
	}
	if cfg.ShutdownTimeout != 7*time.Second {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsMissingFileServiceURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://knowledge:knowledge@localhost:5432/knowledge?sslmode=disable")
	t.Setenv("KNOWLEDGE_REDIS_ADDR", "localhost:6379")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil")
	}
	if !strings.Contains(err.Error(), "FILE_SERVICE_BASE_URL") {
		t.Fatalf("error = %v", err)
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DATABASE_URL",
		"FILE_SERVICE_BASE_URL",
		"KNOWLEDGE_REDIS_ADDR",
		"KNOWLEDGE_SERVICE_TOKEN",
		"KNOWLEDGE_MAX_UPLOAD_BYTES",
		"KNOWLEDGE_HTTP_ADDR",
		"KNOWLEDGE_SERVICE_VERSION",
		"KNOWLEDGE_ENV",
		"KNOWLEDGE_SHUTDOWN_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
}
