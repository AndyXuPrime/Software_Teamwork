package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", "")
	t.Setenv("GATEWAY_SERVICE_VERSION", "")
	t.Setenv("GATEWAY_ENV", "")
	t.Setenv("GATEWAY_MAX_BODY_BYTES", "")
	t.Setenv("GATEWAY_REQUEST_TIMEOUT", "")
	t.Setenv("GATEWAY_SHUTDOWN_TIMEOUT", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_ORIGINS", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_METHODS", "")
	t.Setenv("GATEWAY_CORS_ALLOWED_HEADERS", "")
	t.Setenv("GATEWAY_CORS_ALLOW_CREDENTIALS", "")
	t.Setenv("GATEWAY_DOWNSTREAM_TIMEOUT", "")
	t.Setenv("GATEWAY_REDIS_ADDR", "")
	t.Setenv("GATEWAY_REDIS_PASSWORD", "")
	t.Setenv("GATEWAY_REDIS_DB", "")
	t.Setenv("GATEWAY_TOKEN_HASH_SECRET", "")
	t.Setenv("GATEWAY_TOKEN_HASH_KEY_VERSION", "")
	t.Setenv("GATEWAY_INTERNAL_SERVICE_TOKEN", "")
	t.Setenv("GATEWAY_AUTH_BASE_URL", "")
	t.Setenv("GATEWAY_KNOWLEDGE_BASE_URL", "")
	t.Setenv("GATEWAY_QA_BASE_URL", "")
	t.Setenv("GATEWAY_DOCUMENT_BASE_URL", "")
	t.Setenv("GATEWAY_AI_GATEWAY_BASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.MaxBodyBytes != DefaultMaxBodyBytes {
		t.Fatalf("MaxBodyBytes = %d", cfg.MaxBodyBytes)
	}
	if cfg.RequestTimeout != DefaultRequestTimeout {
		t.Fatalf("RequestTimeout = %s", cfg.RequestTimeout)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "*" {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
	if cfg.RedisAddr != DefaultRedisAddr || cfg.TokenHashKeyVersion != DefaultTokenKeyVersion {
		t.Fatalf("session config = %+v", cfg)
	}
}

func TestLoadParsesEnvironment(t *testing.T) {
	t.Setenv("GATEWAY_HTTP_ADDR", ":9090")
	t.Setenv("GATEWAY_SERVICE_VERSION", "1.2.3")
	t.Setenv("GATEWAY_ENV", "test")
	t.Setenv("GATEWAY_MAX_BODY_BYTES", "2048")
	t.Setenv("GATEWAY_REQUEST_TIMEOUT", "5s")
	t.Setenv("GATEWAY_SHUTDOWN_TIMEOUT", "2s")
	t.Setenv("GATEWAY_CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://example.com")
	t.Setenv("GATEWAY_CORS_ALLOWED_METHODS", "get,post")
	t.Setenv("GATEWAY_CORS_ALLOWED_HEADERS", "Authorization, X-Request-Id")
	t.Setenv("GATEWAY_CORS_ALLOW_CREDENTIALS", "true")
	t.Setenv("GATEWAY_DOWNSTREAM_TIMEOUT", "3s")
	t.Setenv("GATEWAY_REDIS_ADDR", "redis:6379")
	t.Setenv("GATEWAY_REDIS_PASSWORD", "secret")
	t.Setenv("GATEWAY_REDIS_DB", "2")
	t.Setenv("GATEWAY_TOKEN_HASH_SECRET", "hash-secret")
	t.Setenv("GATEWAY_TOKEN_HASH_KEY_VERSION", "v9")
	t.Setenv("GATEWAY_INTERNAL_SERVICE_TOKEN", "svc-token")
	t.Setenv("GATEWAY_AUTH_BASE_URL", "http://auth:8001")
	t.Setenv("GATEWAY_KNOWLEDGE_BASE_URL", "http://knowledge:8002")
	t.Setenv("GATEWAY_QA_BASE_URL", "http://qa:8003")
	t.Setenv("GATEWAY_DOCUMENT_BASE_URL", "http://document:8004")
	t.Setenv("GATEWAY_AI_GATEWAY_BASE_URL", "http://ai-gateway:8005")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":9090" || cfg.ServiceVersion != "1.2.3" || cfg.Environment != "test" {
		t.Fatalf("basic config = %+v", cfg)
	}
	if cfg.MaxBodyBytes != 2048 || cfg.RequestTimeout != 5*time.Second || cfg.ShutdownTimeout != 2*time.Second {
		t.Fatalf("numeric config = %+v", cfg)
	}
	if cfg.DownstreamTimeout != 3*time.Second || cfg.RedisDB != 2 {
		t.Fatalf("downstream config = %+v", cfg)
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "secret" || cfg.TokenHashSecret != "hash-secret" || cfg.TokenHashKeyVersion != "v9" || cfg.InternalServiceToken != "svc-token" {
		t.Fatalf("session config = %+v", cfg)
	}
	if cfg.AuthBaseURL != "http://auth:8001" || cfg.KnowledgeBaseURL != "http://knowledge:8002" || cfg.QABaseURL != "http://qa:8003" || cfg.DocumentBaseURL != "http://document:8004" || cfg.AIGatewayBaseURL != "http://ai-gateway:8005" {
		t.Fatalf("base URLs = %+v", cfg)
	}
	if got, want := cfg.CORSAllowedOrigins[1], "https://example.com"; got != want {
		t.Fatalf("origin = %q, want %q", got, want)
	}
	if !cfg.CORSAllowCredentials {
		t.Fatal("CORSAllowCredentials = false")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
	}{
		{name: "max body", key: "GATEWAY_MAX_BODY_BYTES", val: "0"},
		{name: "request timeout", key: "GATEWAY_REQUEST_TIMEOUT", val: "-1s"},
		{name: "shutdown timeout", key: "GATEWAY_SHUTDOWN_TIMEOUT", val: "bad"},
		{name: "downstream timeout", key: "GATEWAY_DOWNSTREAM_TIMEOUT", val: "0s"},
		{name: "redis db", key: "GATEWAY_REDIS_DB", val: "-1"},
		{name: "cors credentials", key: "GATEWAY_CORS_ALLOW_CREDENTIALS", val: "maybe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.val)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil")
			}
		})
	}
}
