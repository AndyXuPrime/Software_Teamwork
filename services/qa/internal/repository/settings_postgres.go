package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/repository/sqlc"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func (r *Postgres) GetActiveQAConfig(ctx context.Context) (service.RetrievalSettings, []string, error) {
	row, err := r.queries.GetActiveQAConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.RetrievalSettings{TopK: 5, ScoreThreshold: 0.7, RerankThreshold: 0.5, RerankTopN: 3}, []string{}, nil
	}
	if err != nil {
		return service.RetrievalSettings{}, nil, fmt.Errorf("get active QA config: %w", err)
	}
	settings, err := retrievalSettingsFromRow(row)
	if err != nil {
		return service.RetrievalSettings{}, nil, err
	}
	configID, err := uuidFromString(row.ID)
	if err != nil {
		return service.RetrievalSettings{}, nil, err
	}
	ids, err := r.queries.ListQAConfigKnowledgeBaseIDs(ctx, configID)
	if err != nil {
		return service.RetrievalSettings{}, nil, fmt.Errorf("list QA config knowledge bases: %w", err)
	}
	return settings, ids, nil
}

func (r *Postgres) CreateQAConfigVersion(ctx context.Context, userID string, settings service.RetrievalSettings, knowledgeBaseIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin QA config update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if err := q.LockQAConfigVersions(ctx); err != nil {
		return fmt.Errorf("lock QA config versions: %w", err)
	}
	version, err := q.NextQAConfigVersionNo(ctx)
	if err != nil {
		return fmt.Errorf("next QA config version: %w", err)
	}
	if err := q.DeactivateAllQAConfigs(ctx); err != nil {
		return fmt.Errorf("deactivate QA config: %w", err)
	}
	params, err := insertQAConfigVersionParams(settings, version, userID)
	if err != nil {
		return err
	}
	configID, err := q.InsertQAConfigVersion(ctx, params)
	if err != nil {
		return fmt.Errorf("insert QA config version: %w", err)
	}
	parsedConfigID, err := uuidFromString(configID)
	if err != nil {
		return err
	}
	for index, id := range knowledgeBaseIDs {
		if err := q.InsertQAConfigKnowledgeBase(ctx, sqlc.InsertQAConfigKnowledgeBaseParams{
			ConfigID:     parsedConfigID,
			ExternalKbID: id,
			SortOrder:    int32(index),
		}); err != nil {
			return fmt.Errorf("insert QA config knowledge base: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit QA config update: %w", err)
	}
	return nil
}

func (r *Postgres) GetActiveLLMConfig(ctx context.Context) (service.StoredLLMConfig, error) {
	row, err := r.queries.GetActiveLLMConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.StoredLLMConfig{}, service.NewError(service.CodeNotFound, "active LLM configuration not found", err)
	}
	if err != nil {
		return service.StoredLLMConfig{}, fmt.Errorf("get active LLM config: %w", err)
	}
	return storedLLMFromRow(row)
}

func (r *Postgres) CreateLLMConfigVersion(ctx context.Context, userID string, config service.StoredLLMConfig) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin LLM config update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if err := q.LockLLMConfigVersions(ctx); err != nil {
		return fmt.Errorf("lock LLM config versions: %w", err)
	}
	version, err := q.NextLLMConfigVersionNo(ctx)
	if err != nil {
		return fmt.Errorf("next LLM config version: %w", err)
	}
	if err := q.DeactivateAllLLMConfigs(ctx); err != nil {
		return fmt.Errorf("deactivate LLM config: %w", err)
	}
	params, err := insertLLMConfigVersionParams(config, version, userID)
	if err != nil {
		return err
	}
	if err := q.InsertLLMConfigVersion(ctx, params); err != nil {
		return fmt.Errorf("insert LLM config version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit LLM config update: %w", err)
	}
	return nil
}

func (r *Postgres) GetRuntimeSetting(ctx context.Context, key string) (string, error) {
	value, err := r.queries.GetRuntimeSetting(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", service.NewError(service.CodeNotFound, "runtime setting not found", err)
	}
	if err != nil {
		return "", fmt.Errorf("get runtime setting: %w", err)
	}
	return value, nil
}

func (r *Postgres) UpsertRuntimeSetting(ctx context.Context, key, value string) error {
	if err := r.queries.UpsertRuntimeSetting(ctx, sqlc.UpsertRuntimeSettingParams{Key: key, Value: value}); err != nil {
		return fmt.Errorf("upsert runtime setting: %w", err)
	}
	return nil
}

func (r *Postgres) ListMCPServers(ctx context.Context) ([]service.MCPServerRecord, error) {
	rows, err := r.queries.ListMCPServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list MCP servers: %w", err)
	}
	servers := make([]service.MCPServerRecord, 0, len(rows))
	for _, row := range rows {
		server, err := mcpServerFromListRow(row)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func (r *Postgres) GetMCPServer(ctx context.Context, id string) (service.MCPServerRecord, error) {
	parsedID, err := uuidFromString(id)
	if err != nil {
		return service.MCPServerRecord{}, err
	}
	row, err := r.queries.GetMCPServer(ctx, parsedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.MCPServerRecord{}, service.NewError(service.CodeNotFound, "MCP server not found", err)
	}
	if err != nil {
		return service.MCPServerRecord{}, fmt.Errorf("get MCP server: %w", err)
	}
	return mcpServerFromGetRow(row)
}

func (r *Postgres) CreateMCPServer(ctx context.Context, server service.MCPServerRecord) (service.MCPServerRecord, error) {
	params, err := insertMCPServerParams(server)
	if err != nil {
		return service.MCPServerRecord{}, err
	}
	row, err := r.queries.InsertMCPServer(ctx, params)
	if err != nil {
		return service.MCPServerRecord{}, fmt.Errorf("insert MCP server: %w", err)
	}
	server.ID = row.ID
	if row.CreatedAt.Valid {
		server.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		server.UpdatedAt = row.UpdatedAt.Time
	}
	return server, nil
}

func (r *Postgres) UpdateMCPServer(ctx context.Context, server service.MCPServerRecord) (service.MCPServerRecord, error) {
	params, err := updateMCPServerParams(server)
	if err != nil {
		return service.MCPServerRecord{}, err
	}
	updatedAt, err := r.queries.UpdateMCPServer(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.MCPServerRecord{}, service.NewError(service.CodeNotFound, "MCP server not found", err)
	}
	if err != nil {
		return service.MCPServerRecord{}, fmt.Errorf("update MCP server: %w", err)
	}
	if updatedAt.Valid {
		server.UpdatedAt = updatedAt.Time
	}
	return server, nil
}

func (r *Postgres) DeleteMCPServer(ctx context.Context, id string) error {
	parsedID, err := uuidFromString(id)
	if err != nil {
		return err
	}
	rowsAffected, err := r.queries.DeleteMCPServer(ctx, parsedID)
	if err != nil {
		return fmt.Errorf("delete MCP server: %w", err)
	}
	if rowsAffected == 0 {
		return service.NewError(service.CodeNotFound, "MCP server not found", nil)
	}
	return nil
}

func (r *Postgres) UpdateMCPConnectionStatus(ctx context.Context, id string, toolCount int, connectedAt *time.Time, lastError string) error {
	parsedID, err := uuidFromString(id)
	if err != nil {
		return err
	}
	if err := r.queries.UpdateMCPConnectionStatus(ctx, sqlc.UpdateMCPConnectionStatusParams{
		ToolCount:       int32(toolCount),
		LastConnectedAt: timestamptzFromTime(connectedAt),
		LastError:       nullableInterface(lastError),
		ID:              parsedID,
	}); err != nil {
		return fmt.Errorf("update MCP connection status: %w", err)
	}
	return nil
}

func (r *Postgres) WriteAuditLog(ctx context.Context, audit service.AuditLog) error {
	beforeJSON, err := json.Marshal(audit.BeforeData)
	if err != nil {
		return fmt.Errorf("encode audit before data: %w", err)
	}
	afterJSON, err := json.Marshal(audit.AfterData)
	if err != nil {
		return fmt.Errorf("encode audit after data: %w", err)
	}
	if err := r.queries.InsertAuditLog(ctx, sqlc.InsertAuditLogParams{
		ExternalUserID: audit.UserID,
		Action:         audit.Action,
		TargetType:     audit.TargetType,
		TargetID:       nullableInterface(audit.TargetID),
		BeforeData:     beforeJSON,
		AfterData:      afterJSON,
		RequestID:      nullableInterface(audit.RequestID),
	}); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
