package config

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestLoadRequiresSensitiveConfig(t *testing.T) {
	t.Setenv("AI_GATEWAY_DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatalf("Load() error = nil, want required database error")
	}
}

func TestLoadValidConfig(t *testing.T) {
	tokenHash := sha256.Sum256([]byte("service-token"))
	t.Setenv("AI_GATEWAY_DATABASE_URL", "postgres://postgres:postgres@localhost/postgres")
	t.Setenv("AI_GATEWAY_SERVICE_TOKEN_HASHES", "sha256:"+hex.EncodeToString(tokenHash[:]))
	t.Setenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF", "local-v1")
	t.Setenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY", "local-development-secret")
	t.Setenv("AI_GATEWAY_DEFAULT_TIMEOUT_MS", "45000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DefaultTimeoutMS != 45000 {
		t.Fatalf("DefaultTimeoutMS = %d, want 45000", cfg.DefaultTimeoutMS)
	}
	if len(cfg.CredentialEncryptionKey) != sha256.Size {
		t.Fatalf("CredentialEncryptionKey length = %d, want %d", len(cfg.CredentialEncryptionKey), sha256.Size)
	}
}
