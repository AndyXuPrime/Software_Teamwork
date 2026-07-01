package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type attachmentScanner interface{ Scan(...any) error }

func (r *Postgres) CreateAttachment(ctx context.Context, a service.SessionAttachment) (service.SessionAttachment, error) {
	const q = `INSERT INTO session_attachments (id, conversation_id, external_user_id, file_ref, filename, content_type, size_bytes, status, error_summary, page_count, chunk_count, expires_at, created_at, updated_at) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	if _, err := r.pool.Exec(ctx, q, a.ID, a.SessionID, a.OwnerUserID, a.FileRef, a.Filename, a.ContentType, a.SizeBytes, a.Status, a.ErrorSummary, a.PageCount, a.ChunkCount, a.ExpiresAt, a.CreatedAt, a.UpdatedAt); err != nil {
		return service.SessionAttachment{}, fmt.Errorf("insert session attachment: %w", err)
	}
	return a, nil
}

func (r *Postgres) ListAttachments(ctx context.Context, userID, sessionID string, opts service.AttachmentListOptions) (service.Page[service.SessionAttachment], error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 20
	}
	if _, err := r.GetConversation(ctx, userID, sessionID); err != nil {
		return service.Page[service.SessionAttachment]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM session_attachments WHERE conversation_id::text=$1 AND external_user_id=$2 AND deleted_at IS NULL`, sessionID, userID).Scan(&total); err != nil {
		return service.Page[service.SessionAttachment]{}, fmt.Errorf("count attachments: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT id::text, conversation_id::text, external_user_id, file_ref, filename, content_type, size_bytes, status, COALESCE(error_summary,''), page_count, chunk_count, expires_at, deleted_at, created_at, updated_at FROM session_attachments WHERE conversation_id::text=$1 AND external_user_id=$2 AND deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, sessionID, userID, opts.PageSize, (opts.Page-1)*opts.PageSize)
	if err != nil {
		return service.Page[service.SessionAttachment]{}, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	items := []service.SessionAttachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return service.Page[service.SessionAttachment]{}, err
		}
		items = append(items, a)
	}
	return service.Page[service.SessionAttachment]{Items: items, Page: opts.Page, PageSize: opts.PageSize, Total: total}, rows.Err()
}

func (r *Postgres) GetAttachment(ctx context.Context, userID, sessionID, attachmentID string) (service.SessionAttachment, error) {
	row := r.pool.QueryRow(ctx, `SELECT id::text, conversation_id::text, external_user_id, file_ref, filename, content_type, size_bytes, status, COALESCE(error_summary,''), page_count, chunk_count, expires_at, deleted_at, created_at, updated_at FROM session_attachments WHERE id::text=$1 AND conversation_id::text=$2 AND external_user_id=$3 AND deleted_at IS NULL`, attachmentID, sessionID, userID)
	a, err := scanAttachment(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return service.SessionAttachment{}, service.NewError(service.CodeNotFound, "attachment not found", err)
	}
	if err != nil {
		return service.SessionAttachment{}, fmt.Errorf("get attachment: %w", err)
	}
	return a, nil
}

func (r *Postgres) SoftDeleteAttachment(ctx context.Context, userID, sessionID, attachmentID string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE session_attachments SET deleted_at=$4, updated_at=$4 WHERE id::text=$1 AND conversation_id::text=$2 AND external_user_id=$3 AND deleted_at IS NULL`, attachmentID, sessionID, userID, now)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return service.NewError(service.CodeNotFound, "attachment not found", nil)
	}
	return nil
}
func (r *Postgres) MarkAttachmentParsing(ctx context.Context, userID, sessionID, attachmentID string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE session_attachments SET status='parsing', error_summary='', updated_at=$4 WHERE id::text=$1 AND conversation_id::text=$2 AND external_user_id=$3 AND deleted_at IS NULL`, attachmentID, sessionID, userID, now)
	return err
}
func (r *Postgres) MarkAttachmentFailed(ctx context.Context, userID, sessionID, attachmentID, summary string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE session_attachments SET status='failed', error_summary=$4, updated_at=$5 WHERE id::text=$1 AND conversation_id::text=$2 AND external_user_id=$3 AND deleted_at IS NULL`, attachmentID, sessionID, userID, sanitizeAttachmentSummary(summary), now)
	return err
}

func (r *Postgres) ReplaceAttachmentChunks(ctx context.Context, userID, sessionID, attachmentID string, chunks []service.SessionAttachmentChunk, pageCount int, now time.Time) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM session_attachment_chunks WHERE attachment_id::text=$1`, attachmentID); err != nil {
		return err
	}
	for _, c := range chunks {
		if _, err := tx.Exec(ctx, `INSERT INTO session_attachment_chunks (id, attachment_id, conversation_id, chunk_index, page_number, section_path, content, content_preview, token_count, created_at) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10)`, c.ID, attachmentID, sessionID, c.ChunkIndex, c.PageNumber, c.SectionPath, c.Content, c.ContentPreview, c.TokenCount, now); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE session_attachments SET status='ready', error_summary='', page_count=$4, chunk_count=$5, updated_at=$6 WHERE id::text=$1 AND conversation_id::text=$2 AND external_user_id=$3`, attachmentID, sessionID, userID, pageCount, len(chunks), now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Postgres) ValidateReadyAttachments(ctx context.Context, userID, sessionID string, ids []string) ([]service.SessionAttachment, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	out := []service.SessionAttachment{}
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, service.ValidationError(map[string]string{"attachmentIds": "must not contain empty values"})
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		a, err := r.GetAttachment(ctx, userID, sessionID, id)
		if err != nil {
			return nil, err
		}
		if a.Status != service.AttachmentStatusReady {
			return nil, service.ValidationError(map[string]string{"attachmentIds": "attachments must be ready"})
		}
		out = append(out, a)
	}
	return out, nil
}
func (r *Postgres) BindMessageAttachments(ctx context.Context, userID, sessionID, messageID string, ids []string, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := r.ValidateReadyAttachments(ctx, userID, sessionID, ids); err != nil {
		return err
	}
	for _, id := range ids {
		_, err := r.pool.Exec(ctx, `INSERT INTO message_attachments (message_id, attachment_id, created_at) VALUES ($1::uuid,$2::uuid,$3) ON CONFLICT DO NOTHING`, messageID, id, now)
		if err != nil {
			return fmt.Errorf("bind message attachment: %w", err)
		}
	}
	return nil
}

func (r *Postgres) SearchSessionAttachmentChunks(ctx context.Context, userID, sessionID string, attachmentIDs []string, query string, limit int) ([]service.SessionAttachmentChunk, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	args := []any{sessionID, userID, "%" + strings.ToLower(strings.TrimSpace(query)) + "%", limit}
	filter := ""
	if len(attachmentIDs) > 0 {
		filter = " AND a.id::text = ANY($5)"
		args = append(args, attachmentIDs)
	}
	rows, err := r.pool.Query(ctx, `SELECT ch.id::text, ch.attachment_id::text, ch.conversation_id::text, ch.chunk_index, ch.page_number, COALESCE(ch.section_path,''), ch.content, ch.content_preview, ch.token_count, a.filename FROM session_attachment_chunks ch JOIN session_attachments a ON a.id=ch.attachment_id WHERE ch.conversation_id::text=$1 AND a.external_user_id=$2 AND a.status='ready' AND a.deleted_at IS NULL AND lower(ch.content) LIKE $3`+filter+` ORDER BY ch.chunk_index LIMIT $4`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []service.SessionAttachmentChunk
	for rows.Next() {
		var c service.SessionAttachmentChunk
		if err := rows.Scan(&c.ID, &c.AttachmentID, &c.SessionID, &c.ChunkIndex, &c.PageNumber, &c.SectionPath, &c.Content, &c.ContentPreview, &c.TokenCount, &c.Filename); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (r *Postgres) CleanupExpiredAttachments(ctx context.Context, now time.Time, limit int) ([]service.SessionAttachment, error) {
	rows, err := r.pool.Query(ctx, `UPDATE session_attachments SET deleted_at=$1, updated_at=$1 WHERE id IN (SELECT id FROM session_attachments WHERE deleted_at IS NULL AND expires_at <= $1 ORDER BY expires_at LIMIT $2) RETURNING id::text, conversation_id::text, external_user_id, file_ref, filename, content_type, size_bytes, status, COALESCE(error_summary,''), page_count, chunk_count, expires_at, deleted_at, created_at, updated_at`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []service.SessionAttachment
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func scanAttachment(row attachmentScanner) (service.SessionAttachment, error) {
	var a service.SessionAttachment
	err := row.Scan(&a.ID, &a.SessionID, &a.OwnerUserID, &a.FileRef, &a.Filename, &a.ContentType, &a.SizeBytes, &a.Status, &a.ErrorSummary, &a.PageCount, &a.ChunkCount, &a.ExpiresAt, &a.DeletedAt, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}
func sanitizeAttachmentSummary(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) > 200 {
		value = string([]rune(value)[:200])
	}
	return value
}
