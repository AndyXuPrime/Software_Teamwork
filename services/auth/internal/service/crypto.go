package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2MemoryKiB   uint32 = 65536
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 2
	argon2SaltBytes          = 16
	argon2KeyBytes           = 32
	opaqueTokenBytes         = 32
	opaqueTokenPrefix        = "atk_v1_"
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2MemoryKiB, argon2Parallelism, argon2KeyBytes)
	return encodePHC(salt, key), nil
}

func verifyPassword(password string, encoded string) (bool, error) {
	params, salt, expected, err := decodePHC(encoded)
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memoryKiB, params.parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func passwordHashParamsJSON() string {
	return `{"memoryKiB":65536,"iterations":3,"parallelism":2,"saltBytes":16,"keyBytes":32}`
}

func newOpaqueAccessToken() (string, error) {
	bytes := make([]byte, opaqueTokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return opaqueTokenPrefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashAccessToken(accessToken string, secret []byte, keyVersion string) (string, error) {
	trimmed := strings.TrimSpace(accessToken)
	if trimmed == "" {
		return "", fmt.Errorf("access token is required")
	}
	if len(secret) == 0 {
		return "", fmt.Errorf("token hash secret is required")
	}
	keyVersion = strings.TrimSpace(keyVersion)
	if keyVersion == "" {
		keyVersion = TokenHashKeyVersionV1
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(trimmed))
	return TokenHashAlg + ":" + keyVersion + ":" + hex.EncodeToString(mac.Sum(nil)), nil
}

func encodePHC(salt []byte, key []byte) string {
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2MemoryKiB,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

type argon2Params struct {
	memoryKiB   uint32
	iterations  uint32
	parallelism uint8
}

func decodePHC(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash is not a PHC string")
	}
	if parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash algorithm is unsupported")
	}
	if parts[2] != "v=19" {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash version is unsupported")
	}
	params, err := parseArgon2Params(parts[3])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash salt is invalid: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash key is invalid: %w", err)
	}
	if len(salt) != argon2SaltBytes || len(key) != argon2KeyBytes {
		return argon2Params{}, nil, nil, fmt.Errorf("password hash parameters do not match %s", PasswordHashParamsVersion)
	}
	return params, salt, key, nil
}

func parseArgon2Params(raw string) (argon2Params, error) {
	values := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return argon2Params{}, fmt.Errorf("password hash parameters are invalid")
		}
		values[key] = value
	}
	memory, err := parseUint32(values["m"])
	if err != nil {
		return argon2Params{}, fmt.Errorf("password hash memory parameter is invalid")
	}
	iterations, err := parseUint32(values["t"])
	if err != nil {
		return argon2Params{}, fmt.Errorf("password hash iterations parameter is invalid")
	}
	parallelism64, err := strconv.ParseUint(values["p"], 10, 8)
	if err != nil {
		return argon2Params{}, fmt.Errorf("password hash parallelism parameter is invalid")
	}
	params := argon2Params{memoryKiB: memory, iterations: iterations, parallelism: uint8(parallelism64)}
	if params.memoryKiB != argon2MemoryKiB || params.iterations != argon2Iterations || params.parallelism != argon2Parallelism {
		return argon2Params{}, fmt.Errorf("password hash parameters do not match %s", PasswordHashParamsVersion)
	}
	return params, nil
}

func parseUint32(raw string) (uint32, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	return uint32(value), err
}
