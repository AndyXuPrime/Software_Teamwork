package redisstore

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

const sessionKeyPrefix = "gateway:session:"

type Config struct {
	Addr       string
	Username   string
	Password   string
	DB         int
	TLSEnabled bool
}

type SessionStore struct {
	client *redis.Client
}

func New(cfg Config) (*SessionStore, error) {
	opt, err := redisOptions(cfg)
	if err != nil {
		return nil, err
	}
	return &SessionStore{client: redis.NewClient(opt)}, nil
}

func redisOptions(cfg Config) (*redis.Options, error) {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil, fmt.Errorf("redis address must not be empty")
	}
	opt := &redis.Options{
		Addr:     strings.TrimSpace(cfg.Addr),
		Username: strings.TrimSpace(cfg.Username),
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.TLSEnabled {
		opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return opt, nil
}

func (s *SessionStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *SessionStore) CheckReady(ctx context.Context) error {
	if s == nil || s.client == nil {
		return service.ErrSessionStoreUnavailable
	}
	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", service.ErrSessionStoreUnavailable, err)
	}
	return nil
}

func (s *SessionStore) Put(ctx context.Context, entry service.SessionCacheEntry, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return service.ErrSessionStoreUnavailable
	}
	if ttl <= 0 {
		return service.ErrSessionInvalid
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("%w: marshal session cache entry", service.ErrSessionInvalid)
	}
	if err := s.client.Set(ctx, sessionKey(entry.AccessTokenHash), payload, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", service.ErrSessionStoreUnavailable, err)
	}
	return nil
}

func (s *SessionStore) Get(ctx context.Context, accessTokenHash string) (service.SessionCacheEntry, error) {
	if s == nil || s.client == nil {
		return service.SessionCacheEntry{}, service.ErrSessionStoreUnavailable
	}
	value, err := s.client.Get(ctx, sessionKey(accessTokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return service.SessionCacheEntry{}, service.ErrSessionNotFound
	}
	if err != nil {
		return service.SessionCacheEntry{}, fmt.Errorf("%w: %v", service.ErrSessionStoreUnavailable, err)
	}
	var entry service.SessionCacheEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil {
		return service.SessionCacheEntry{}, fmt.Errorf("%w: decode session cache entry", service.ErrSessionInvalid)
	}
	return entry, nil
}

func (s *SessionStore) Delete(ctx context.Context, accessTokenHash string) error {
	if s == nil || s.client == nil {
		return service.ErrSessionStoreUnavailable
	}
	if err := s.client.Del(ctx, sessionKey(accessTokenHash)).Err(); err != nil {
		return fmt.Errorf("%w: %v", service.ErrSessionStoreUnavailable, err)
	}
	return nil
}

func sessionKey(accessTokenHash string) string {
	return sessionKeyPrefix + strings.TrimSpace(accessTokenHash)
}
