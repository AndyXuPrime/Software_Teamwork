package redisstore

import (
	"crypto/tls"
	"testing"
)

func TestRedisOptionsIncludesManagedRedisFields(t *testing.T) {
	opt, err := redisOptions(Config{
		Addr:       " redis.example.com:6380 ",
		Username:   " gateway-user ",
		Password:   "secret",
		DB:         2,
		TLSEnabled: true,
	})
	if err != nil {
		t.Fatalf("redisOptions() error = %v", err)
	}
	if opt.Addr != "redis.example.com:6380" || opt.Username != "gateway-user" || opt.Password != "secret" || opt.DB != 2 {
		t.Fatalf("unexpected options: %+v", opt)
	}
	if opt.TLSConfig == nil || opt.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("TLSConfig = %+v, want TLS 1.2 minimum", opt.TLSConfig)
	}
}

func TestRedisOptionsKeepsLocalDefaults(t *testing.T) {
	opt, err := redisOptions(Config{Addr: "localhost:6379"})
	if err != nil {
		t.Fatalf("redisOptions() error = %v", err)
	}
	if opt.Username != "" || opt.Password != "" || opt.DB != 0 || opt.TLSConfig != nil {
		t.Fatalf("unexpected local defaults: %+v", opt)
	}
}

func TestRedisOptionsRejectsMissingAddress(t *testing.T) {
	if _, err := redisOptions(Config{}); err == nil {
		t.Fatal("redisOptions() error = nil")
	}
}
