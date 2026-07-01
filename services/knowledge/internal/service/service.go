package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Clock func() time.Time

type IDGenerator func(prefix string) string

// Service supports parser-config admin CRUD for adapter mode.
type Service struct {
	repo  ParserConfigRepository
	now   Clock
	newID IDGenerator
}

func New(repo ParserConfigRepository) *Service {
	return &Service{
		repo:  repo,
		now:   func() time.Time { return time.Now().UTC() },
		newID: newID,
	}
}

func NewWithOptions(repo ParserConfigRepository, now Clock, idGenerator IDGenerator) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if idGenerator == nil {
		idGenerator = newID
	}
	return &Service{repo: repo, now: now, newID: idGenerator}
}

func hasAdminRole(roles []string) bool {
	for _, role := range roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "admin", "super_admin", "superadmin":
			return true
		}
	}
	return false
}

func hasPermission(permissions []string, target string) bool {
	for _, permission := range permissions {
		if strings.TrimSpace(permission) == target {
			return true
		}
	}
	return false
}

func repositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotFound) {
		return NotFoundError("resource not found")
	}
	if errors.Is(err, ErrConflict) {
		return ConflictError("resource already exists", err)
	}
	if _, ok := Classify(err); ok {
		return err
	}
	return DependencyError("repository operation failed", err)
}

func newID(prefix string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000"), ".", "")
	}
	return prefix + "_" + hex.EncodeToString(buf[:])
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
