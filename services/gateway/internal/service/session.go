package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionInvalid          = errors.New("session invalid")
	ErrSessionStoreUnavailable = errors.New("session store unavailable")
)

type SessionStore interface {
	Put(ctx context.Context, entry SessionCacheEntry, ttl time.Duration) error
	Get(ctx context.Context, accessTokenHash string) (SessionCacheEntry, error)
	Delete(ctx context.Context, accessTokenHash string) error
}

type TokenHasher struct {
	secret     []byte
	keyVersion string
}

func NewTokenHasher(secret string, keyVersion string) (TokenHasher, error) {
	secret = strings.TrimSpace(secret)
	keyVersion = strings.TrimSpace(keyVersion)
	if secret == "" {
		return TokenHasher{}, fmt.Errorf("token hash secret must not be empty")
	}
	if keyVersion == "" {
		return TokenHasher{}, fmt.Errorf("token hash key version must not be empty")
	}
	return TokenHasher{secret: []byte(secret), keyVersion: keyVersion}, nil
}

func (h TokenHasher) Hash(accessToken string) (string, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", ErrSessionInvalid
	}
	if len(h.secret) == 0 || strings.TrimSpace(h.keyVersion) == "" {
		return "", fmt.Errorf("token hasher is not configured")
	}
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(accessToken))
	return "hmac-sha256:" + h.keyVersion + ":" + hex.EncodeToString(mac.Sum(nil)), nil
}

type UserSummary struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type UserRecord struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

type SessionSummary struct {
	SessionID   string    `json:"sessionId"`
	AccessToken string    `json:"accessToken"`
	TokenType   string    `json:"tokenType"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type SessionIdentity struct {
	SessionID    string      `json:"sessionId"`
	User         UserSummary `json:"user"`
	TokenType    string      `json:"tokenType"`
	ExpiresAt    time.Time   `json:"expiresAt"`
	IssuedAt     time.Time   `json:"issuedAt"`
	RevokedAt    *time.Time  `json:"revokedAt,omitempty"`
	RevokeReason *string     `json:"revokeReason,omitempty"`
}

type SessionResponse struct {
	User    UserSummary    `json:"user"`
	Session SessionSummary `json:"session"`
}

type SessionEnvelope struct {
	Data      SessionResponse `json:"data"`
	RequestID string          `json:"requestId"`
}

type UserEnvelope struct {
	Data      UserSummary `json:"data"`
	RequestID string      `json:"requestId"`
}

type UserRecordEnvelope struct {
	Data      UserRecord `json:"data"`
	RequestID string     `json:"requestId"`
}

type SessionIdentityEnvelope struct {
	Data      SessionIdentity `json:"data"`
	RequestID string          `json:"requestId"`
}

type SessionCacheEntry struct {
	SessionID       string    `json:"sessionId"`
	UserID          string    `json:"userId"`
	Username        string    `json:"username"`
	Roles           []string  `json:"roles"`
	Permissions     []string  `json:"permissions"`
	TokenType       string    `json:"tokenType"`
	AccessTokenHash string    `json:"accessTokenHash"`
	ExpiresAt       time.Time `json:"expiresAt"`
	IssuedAt        time.Time `json:"issuedAt"`
	CachedAt        time.Time `json:"cachedAt"`
	RequestID       string    `json:"requestId"`
}

func CacheEntryFromSession(result SessionResponse, accessTokenHash string, requestID string, now time.Time) (SessionCacheEntry, time.Duration, error) {
	entry := SessionCacheEntry{
		SessionID:       strings.TrimSpace(result.Session.SessionID),
		UserID:          strings.TrimSpace(result.User.ID),
		Username:        strings.TrimSpace(result.User.Username),
		Roles:           safeStrings(result.User.Roles),
		Permissions:     safeStrings(result.User.Permissions),
		TokenType:       strings.TrimSpace(result.Session.TokenType),
		AccessTokenHash: strings.TrimSpace(accessTokenHash),
		ExpiresAt:       result.Session.ExpiresAt,
		IssuedAt:        now,
		CachedAt:        now,
		RequestID:       requestID,
	}
	if entry.TokenType == "" {
		entry.TokenType = "Bearer"
	}
	if err := entry.Validate(accessTokenHash, now); err != nil {
		return SessionCacheEntry{}, 0, err
	}
	return entry, entry.ExpiresAt.Sub(now), nil
}

func CacheEntryFromIdentity(identity SessionIdentity, user UserRecord, accessTokenHash string, requestID string, now time.Time) (SessionCacheEntry, time.Duration, error) {
	if identity.RevokedAt != nil || !identity.ExpiresAt.After(now) || !strings.EqualFold(strings.TrimSpace(user.Status), "active") {
		return SessionCacheEntry{}, 0, ErrSessionInvalid
	}
	entry := SessionCacheEntry{
		SessionID:       strings.TrimSpace(identity.SessionID),
		UserID:          strings.TrimSpace(user.ID),
		Username:        strings.TrimSpace(user.Username),
		Roles:           safeStrings(user.Roles),
		Permissions:     safeStrings(user.Permissions),
		TokenType:       strings.TrimSpace(identity.TokenType),
		AccessTokenHash: strings.TrimSpace(accessTokenHash),
		ExpiresAt:       identity.ExpiresAt,
		IssuedAt:        identity.IssuedAt,
		CachedAt:        now,
		RequestID:       requestID,
	}
	if entry.TokenType == "" {
		entry.TokenType = "Bearer"
	}
	if entry.UserID == "" {
		entry.UserID = strings.TrimSpace(identity.User.ID)
	}
	if entry.Username == "" {
		entry.Username = strings.TrimSpace(identity.User.Username)
	}
	if entry.Roles == nil {
		entry.Roles = safeStrings(identity.User.Roles)
	}
	if entry.Permissions == nil {
		entry.Permissions = safeStrings(identity.User.Permissions)
	}
	if err := entry.Validate(accessTokenHash, now); err != nil {
		return SessionCacheEntry{}, 0, err
	}
	return entry, entry.ExpiresAt.Sub(now), nil
}

func (e SessionCacheEntry) Validate(accessTokenHash string, now time.Time) error {
	if strings.TrimSpace(e.SessionID) == "" ||
		strings.TrimSpace(e.UserID) == "" ||
		strings.TrimSpace(e.Username) == "" ||
		strings.TrimSpace(e.AccessTokenHash) == "" ||
		e.ExpiresAt.IsZero() {
		return ErrSessionInvalid
	}
	if accessTokenHash != "" && e.AccessTokenHash != accessTokenHash {
		return ErrSessionInvalid
	}
	if !e.ExpiresAt.After(now) {
		return ErrSessionInvalid
	}
	if e.Roles == nil || e.Permissions == nil {
		return ErrSessionInvalid
	}
	return nil
}

func (e SessionCacheEntry) UserSummary() UserSummary {
	return UserSummary{
		ID:          e.UserID,
		Username:    e.Username,
		Roles:       safeStrings(e.Roles),
		Permissions: safeStrings(e.Permissions),
	}
}

func safeStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
