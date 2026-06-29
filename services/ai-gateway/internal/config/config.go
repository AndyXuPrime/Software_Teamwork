package config

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultHTTPAddr        = ":8086"
	DefaultSecretMode      = "encrypted_column"
	DefaultTimeoutMS       = 60000
	DefaultMaxRequestBytes = int64(1 << 20)
	DefaultShutdownTimeout = 10 * time.Second
	ServiceTokenHashPrefix = "sha256:"
	credentialKeyMinLength = 16
)

type Config struct {
	HTTPAddr                   string
	DatabaseURL                string
	ServiceTokenHashes         []string
	SecretMode                 string
	CredentialEncryptionKeyRef string
	CredentialEncryptionKey    []byte
	DefaultTimeoutMS           int
	MaxRequestBytes            int64
	MetricsAddr                string
	ShutdownTimeout            time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:                   envOr("AI_GATEWAY_HTTP_ADDR", DefaultHTTPAddr),
		DatabaseURL:                strings.TrimSpace(os.Getenv("AI_GATEWAY_DATABASE_URL")),
		ServiceTokenHashes:         splitCSV(os.Getenv("AI_GATEWAY_SERVICE_TOKEN_HASHES")),
		SecretMode:                 envOr("AI_GATEWAY_SECRET_MODE", DefaultSecretMode),
		CredentialEncryptionKeyRef: strings.TrimSpace(os.Getenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF")),
		DefaultTimeoutMS:           DefaultTimeoutMS,
		MaxRequestBytes:            DefaultMaxRequestBytes,
		MetricsAddr:                strings.TrimSpace(os.Getenv("AI_GATEWAY_METRICS_ADDR")),
		ShutdownTimeout:            DefaultShutdownTimeout,
	}

	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY")); raw != "" {
		if len(raw) < credentialKeyMinLength {
			return Config{}, fmt.Errorf("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY must be at least %d characters", credentialKeyMinLength)
		}
		sum := sha256.Sum256([]byte(raw))
		cfg.CredentialEncryptionKey = sum[:]
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_DEFAULT_TIMEOUT_MS")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1000 {
			return Config{}, errors.New("AI_GATEWAY_DEFAULT_TIMEOUT_MS must be an integer >= 1000")
		}
		cfg.DefaultTimeoutMS = value
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_MAX_REQUEST_BYTES")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			return Config{}, errors.New("AI_GATEWAY_MAX_REQUEST_BYTES must be a positive integer")
		}
		cfg.MaxRequestBytes = value
	}
	if raw := strings.TrimSpace(os.Getenv("AI_GATEWAY_SHUTDOWN_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return Config{}, errors.New("AI_GATEWAY_SHUTDOWN_TIMEOUT must be a positive duration")
		}
		cfg.ShutdownTimeout = value
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return errors.New("AI_GATEWAY_HTTP_ADDR is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("AI_GATEWAY_DATABASE_URL is required")
	}
	if len(c.ServiceTokenHashes) == 0 {
		return errors.New("AI_GATEWAY_SERVICE_TOKEN_HASHES is required")
	}
	for _, hash := range c.ServiceTokenHashes {
		if err := validateServiceTokenHash(hash); err != nil {
			return err
		}
	}
	if c.SecretMode != DefaultSecretMode {
		return fmt.Errorf("AI_GATEWAY_SECRET_MODE=%q is not implemented; supported values: %s", c.SecretMode, DefaultSecretMode)
	}
	if strings.TrimSpace(c.CredentialEncryptionKeyRef) == "" {
		return errors.New("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY_REF is required when AI_GATEWAY_SECRET_MODE=encrypted_column")
	}
	if len(c.CredentialEncryptionKey) != sha256.Size {
		return errors.New("AI_GATEWAY_CREDENTIAL_ENCRYPTION_KEY is required when AI_GATEWAY_SECRET_MODE=encrypted_column")
	}
	if c.DefaultTimeoutMS < 1000 {
		return errors.New("AI_GATEWAY_DEFAULT_TIMEOUT_MS must be >= 1000")
	}
	if c.MaxRequestBytes <= 0 {
		return errors.New("AI_GATEWAY_MAX_REQUEST_BYTES must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("AI_GATEWAY_SHUTDOWN_TIMEOUT must be positive")
	}
	return nil
}

func validateServiceTokenHash(value string) error {
	if !strings.HasPrefix(value, ServiceTokenHashPrefix) {
		return errors.New("AI_GATEWAY_SERVICE_TOKEN_HASHES entries must use sha256:<hex>")
	}
	hexValue := strings.TrimPrefix(value, ServiceTokenHashPrefix)
	if len(hexValue) != sha256.Size*2 {
		return errors.New("AI_GATEWAY_SERVICE_TOKEN_HASHES entries must be sha256 hashes")
	}
	for _, r := range hexValue {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return errors.New("AI_GATEWAY_SERVICE_TOKEN_HASHES entries must be hexadecimal")
		}
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
