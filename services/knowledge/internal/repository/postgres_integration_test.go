package repository_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/repository"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

func TestPostgresRepositoryParserConfigLifecycle(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("KNOWLEDGE_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("KNOWLEDGE_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS parser_configs (
		id text PRIMARY KEY,
		name text NOT NULL,
		backend text NOT NULL,
		enabled boolean NOT NULL DEFAULT true,
		is_default boolean NOT NULL DEFAULT false,
		concurrency integer NOT NULL DEFAULT 1,
		supported_content_types text[] NOT NULL DEFAULT '{}',
		endpoint_url text,
		default_parameters jsonb NOT NULL DEFAULT '{}',
		created_at timestamptz NOT NULL,
		updated_at timestamptz NOT NULL,
		deleted_at timestamptz
	)`); err != nil {
		t.Fatalf("ensure parser_configs table: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS parser_config_audits (
		id text PRIMARY KEY,
		parser_config_id text NOT NULL,
		actor_user_id text NOT NULL,
		action text NOT NULL,
		summary jsonb NOT NULL DEFAULT '{}',
		created_at timestamptz NOT NULL
	)`); err != nil {
		t.Fatalf("ensure parser_config_audits table: %v", err)
	}

	repo := repository.NewPostgresRepository(pool)
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	config := service.ParserConfig{
		ID:                    "parser_config_it_1",
		Name:                  "integration-default",
		Backend:               service.ParserBackendBuiltin,
		Enabled:               true,
		IsDefault:             true,
		Concurrency:           2,
		SupportedContentTypes: []string{"text/plain"},
		DefaultParameters:     json.RawMessage(`{}`),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	audit := service.ParserConfigAudit{
		ID:             "audit_it_1",
		ParserConfigID: config.ID,
		ActorUserID:    "usr_admin",
		Action:         "created",
		Summary:        json.RawMessage(`{"changed":["configuration"]}`),
		CreatedAt:      now,
	}

	created, err := repo.CreateParserConfig(ctx, config, audit)
	if err != nil {
		t.Fatalf("CreateParserConfig() error = %v", err)
	}
	if created.ID != config.ID {
		t.Fatalf("created id = %q", created.ID)
	}

	got, err := repo.GetParserConfig(ctx, config.ID)
	if err != nil {
		t.Fatalf("GetParserConfig() error = %v", err)
	}
	if got.Name != config.Name {
		t.Fatalf("GetParserConfig() name = %q", got.Name)
	}

	t.Cleanup(func() {
		_ = repo.SoftDeleteParserConfig(context.Background(), config.ID, now, service.ParserConfigAudit{
			ID: "audit_it_cleanup", ParserConfigID: config.ID, ActorUserID: "usr_admin",
			Action: "disabled", Summary: json.RawMessage(`{}`), CreatedAt: now,
		})
	})
}
