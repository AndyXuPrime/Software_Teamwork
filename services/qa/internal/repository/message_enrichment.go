package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func (r *Postgres) enrichMessages(ctx context.Context, userID, conversationID string, messages []service.Message, options service.MessageListOptions) error {
	if len(messages) == 0 {
		return nil
	}
	ids := messageIDs(messages)
	if options.IncludeThinking {
		steps, err := r.listThinkingForMessages(ctx, userID, conversationID, ids)
		if err != nil {
			return err
		}
		for index := range messages {
			messages[index].Thinking = steps[messages[index].ID]
		}
	}
	if options.IncludeCitations {
		citations, err := r.listCitationsForMessages(ctx, userID, conversationID, ids)
		if err != nil {
			return err
		}
		for index := range messages {
			messages[index].Citations = citations[messages[index].ID]
		}
	}
	return nil
}

func messageIDs(messages []service.Message) []string {
	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}
	return ids
}

func (r *Postgres) listThinkingForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string][]service.ReasoningStep, error) {
	rows, err := r.pool.Query(ctx, `
SELECT
    rr.assistant_message_id::text,
    ps.id::text,
    ps.step_type,
    COALESCE(ps.label, ''),
    COALESCE(ps.detail, ''),
    ps.status,
    ps.created_at
FROM response_process_steps ps
JOIN response_runs rr ON rr.id = ps.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id::text = ANY($1::text[])
    AND rr.conversation_id::text = $2
    AND c.external_user_id = $3
    AND c.deleted_at IS NULL
ORDER BY rr.assistant_message_id::text, ps.step_order`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message thinking: %w", err)
	}
	defer rows.Close()

	items := map[string][]service.ReasoningStep{}
	for rows.Next() {
		var messageID string
		var step service.ReasoningStep
		var label, detail sql.NullString
		if err := rows.Scan(&messageID, &step.ID, &step.Type, &label, &detail, &step.Status, &step.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message thinking: %w", err)
		}
		step.MessageID = messageID
		if label.Valid {
			step.Title = label.String
		}
		if detail.Valid {
			step.Summary = detail.String
		}
		items[messageID] = append(items[messageID], step)
	}
	return items, rows.Err()
}

func (r *Postgres) listCitationsForMessages(ctx context.Context, userID, conversationID string, messageIDs []string) (map[string][]service.Citation, error) {
	rows, err := r.pool.Query(ctx, messageCitationSelect+` WHERE ci.message_id::text = ANY($1::text[]) AND m.conversation_id::text = $2 AND c.external_user_id = $3 AND c.deleted_at IS NULL ORDER BY ci.message_id, ci.citation_no`, messageIDs, conversationID, userID)
	if err != nil {
		return nil, fmt.Errorf("list message citations: %w", err)
	}
	defer rows.Close()
	citations, err := scanCitations(rows)
	if err != nil {
		return nil, err
	}
	items := map[string][]service.Citation{}
	for _, citation := range citations {
		items[citation.MessageID] = append(items[citation.MessageID], citation)
	}
	return items, nil
}
