package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("AUTH_HTTP_ADDR", "")
	t.Setenv("AUTH_SERVICE_VERSION", "")
	t.Setenv("AUTH_ENV", "")
	t.Setenv("AUTH_DATABASE_URL", "")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "")
	t.Setenv("AUTH_TOKEN_HASH_KEY_VERSION", "")
	t.Setenv("AUTH_SESSION_TTL", "")
	t.Setenv("AUTH_DEFAULT_ROLE_CODE", "")
	t.Setenv("AUTH_SHUTDOWN_TIMEOUT", "")
	t.Setenv("AUTH_READINESS_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.ServiceVersion != DefaultServiceVersion {
		t.Fatalf("ServiceVersion = %q", cfg.ServiceVersion)
	}
	if cfg.Environment != DefaultEnvironment {
		t.Fatalf("Environment = %q", cfg.Environment)
	}
	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if cfg.ReadinessTimeout != DefaultReadinessTimeout {
		t.Fatalf("ReadinessTimeout = %s", cfg.ReadinessTimeout)
	}
	if cfg.SessionTTL != DefaultSessionTTL {
		t.Fatalf("SessionTTL = %s", cfg.SessionTTL)
	}
	if cfg.TokenKeyVersion != DefaultTokenKeyVersion || cfg.DefaultRoleCode != DefaultRoleCode {
		t.Fatalf("token/role defaults = %+v", cfg)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("AUTH_HTTP_ADDR", ":9100")
	t.Setenv("AUTH_SERVICE_VERSION", "0.2.0")
	t.Setenv("AUTH_ENV", "test")
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "test-service-token")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_TOKEN_HASH_KEY_VERSION", "v9")
	t.Setenv("AUTH_SESSION_TTL", "2h")
	t.Setenv("AUTH_DEFAULT_ROLE_CODE", "member")
	t.Setenv("AUTH_SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("AUTH_READINESS_TIMEOUT", "3")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":9100" || cfg.ServiceVersion != "0.2.0" || cfg.Environment != "test" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.DatabaseURL == "" {
		t.Fatalf("DatabaseURL is empty")
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
	if cfg.ReadinessTimeout != 3*time.Second {
		t.Fatalf("ReadinessTimeout = %s", cfg.ReadinessTimeout)
	}
	if cfg.ServiceToken != "test-service-token" || cfg.TokenHashSecret != "test-token-hash-secret" || cfg.TokenKeyVersion != "v9" || cfg.DefaultRoleCode != "member" {
		t.Fatalf("auth config = %+v", cfg)
	}
	if cfg.SessionTTL != 2*time.Hour {
		t.Fatalf("SessionTTL = %s", cfg.SessionTTL)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv("AUTH_SHUTDOWN_TIMEOUT", "nope")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

func TestLoadRequiresTokenSecretWhenDatabaseConfigured(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "test-service-token")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}

func TestLoadRequiresServiceTokenWhenDatabaseConfigured(t *testing.T) {
	t.Setenv("AUTH_DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	t.Setenv("AUTH_TOKEN_HASH_SECRET", "test-token-hash-secret")
	t.Setenv("AUTH_INTERNAL_SERVICE_TOKEN", "")

	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil")
	}
}
