package worker

import (
	"crypto/tls"
	"testing"
)

func TestRedisClientOptMapsCloudRedisConfig(t *testing.T) {
	opt := redisClientOpt(RedisConfig{
		Addr:       "redis.example.test:6380",
		Username:   "document",
		Password:   "secret",
		DB:         7,
		TLSEnabled: true,
	})

	if opt.Addr != "redis.example.test:6380" {
		t.Fatalf("Addr = %q", opt.Addr)
	}
	if opt.Username != "document" || opt.Password != "secret" || opt.DB != 7 {
		t.Fatalf("unexpected auth/db options: %+v", opt)
	}
	if opt.TLSConfig == nil {
		t.Fatal("TLSConfig should be set")
	}
	if opt.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLS MinVersion = %v, want TLS 1.2", opt.TLSConfig.MinVersion)
	}
}

func TestRedisClientOptKeepsLocalDefaults(t *testing.T) {
	opt := redisClientOpt(RedisConfig{Addr: "localhost:6379"})

	if opt.Addr != "localhost:6379" {
		t.Fatalf("Addr = %q", opt.Addr)
	}
	if opt.Username != "" || opt.Password != "" || opt.DB != 0 {
		t.Fatalf("local defaults should be empty auth and db 0: %+v", opt)
	}
	if opt.TLSConfig != nil {
		t.Fatal("TLSConfig should be nil by default")
	}
}
