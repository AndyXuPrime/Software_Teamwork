package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestServiceTokenAuthenticator(t *testing.T) {
	sum := sha256.Sum256([]byte("service-token"))
	auth, err := NewServiceTokenAuthenticator([]string{"sha256:" + hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatalf("NewServiceTokenAuthenticator() error = %v", err)
	}
	if !auth.Authenticate("service-token") {
		t.Fatalf("expected token to authenticate")
	}
	if auth.Authenticate("wrong-token") {
		t.Fatalf("expected wrong token to fail")
	}
}
