package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

const ServiceTokenHashPrefix = "sha256:"

type ServiceTokenAuthenticator struct {
	hashes [][]byte
}

func NewServiceTokenAuthenticator(values []string) (*ServiceTokenAuthenticator, error) {
	auth := &ServiceTokenAuthenticator{hashes: make([][]byte, 0, len(values))}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		trimmed = strings.TrimPrefix(trimmed, ServiceTokenHashPrefix)
		decoded, err := hex.DecodeString(trimmed)
		if err != nil || len(decoded) != sha256.Size {
			return nil, errInvalidHash
		}
		auth.hashes = append(auth.hashes, decoded)
	}
	if len(auth.hashes) == 0 {
		return nil, errInvalidHash
	}
	return auth, nil
}

func (a *ServiceTokenAuthenticator) Authenticate(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || a == nil {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	for _, expected := range a.hashes {
		if subtle.ConstantTimeCompare(sum[:], expected) == 1 {
			return true
		}
	}
	return false
}

type invalidHashError struct{}

func (invalidHashError) Error() string { return "invalid service token hash" }

var errInvalidHash = invalidHashError{}
