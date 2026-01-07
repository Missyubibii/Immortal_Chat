// Package repository implements data persistence adapters
// Following Hexagonal Architecture: Adapters implement ports defined in core
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"immortal-chat/internal/core/domain"
	"immortal-chat/internal/core/ports"
)

// Ensure MariaDBRepository implements the required interfaces
var (
	_ ports.WebhookRepository      = (*MariaDBRepository)(nil)
	_ ports.MessageRepository      = (*MariaDBRepository)(nil)
	_ ports.ConversationRepository = (*MariaDBRepository)(nil)
)

// MariaDBRepository implements persistence operations for MariaDB
// Updated to match new schema from technical specification document
type MariaDBRepository struct {
	db *sql.DB
}

// NewMariaDBRepository creates a new MariaDB repository instance
func NewMariaDBRepository(db *sql.DB) *MariaDBRepository {
	return &MariaDBRepository{
		db: db,
	}
}

// ============================================================================
// WebhookRepository Implementation
// ============================================================================

// SaveLog persists a webhook event to the audit log
// Updated for new schema: payload_json (JSON type)
func (r *MariaDBRepository) SaveLog(ctx context.Context, log *domain.WebhookLog) error {
	query := `
		INSERT INTO webhook_logs (platform, payload_json, status, retry_count, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		log.Platform,
		log.PayloadJSON,
		log.Status,
		log.RetryCount,
		log.CreatedAt,
	)
	
	if err != nil {
		slog.Error("Failed to save webhook log",
			"error", err,
			"platform", log.Platform,
		)
		return fmt.Errorf("save webhook log: %w", err)
	}
	
	slog.Debug("Webhook log saved",
		"platform", log.Platform,
		"status", log.Status,
	)
	
	return nil
}

// UpdateStatus updates the processing status of a webhook log
func (r *MariaDBRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE webhook_logs
		SET status = ?
		WHERE id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		slog.Error("Failed to update webhook status",
			"error", err,
			"webhook_id", id,
			"status", status,
		)
		return fmt.Errorf("update webhook status: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		slog.Warn("No webhook log found for status update",
			"webhook_id", id,
		)
	}
	
	return nil
}

// ============================================================================
// MessageRepository Implementation
// ============================================================================

// SaveMessage persists a parsed message to the database
// Updated for new schema: sender_type, type, external_msg_id, attachments (JSON)
func (r *MariaDBRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (
			conversation_id, sender_id, sender_type, content, 
			attachments, type, is_synced, external_msg_id, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			content = VALUES(content)
	`
	
	_, err := r.db.ExecContext(ctx, query,
		msg.ConversationID,
		msg.SenderID,
		msg.SenderType,
		msg.Content,
		msg.Attachments,
		msg.Type,
		msg.IsSynced,
		msg.ExternalMsgID,
		msg.CreatedAt,
	)
	
	if err != nil {
		slog.Error("Failed to save message",
			"error", err,
			"conversation_id", msg.ConversationID,
		)
		return fmt.Errorf("save message: %w", err)
	}
	
	slog.Info("Message saved successfully",
		"conversation_id", msg.ConversationID,
		"sender_type", msg.SenderType,
	)
	
	return nil
}

// GetByID retrieves a message by its database ID
func (r *MariaDBRepository) GetByID(ctx context.Context, id string) (*domain.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, sender_type, content, 
			   attachments, type, is_synced, external_msg_id, created_at
		FROM messages
		WHERE id = ?
	`
	
	var msg domain.Message
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderID,
		&msg.SenderType,
		&msg.Content,
		&msg.Attachments,
		&msg.Type,
		&msg.IsSynced,
		&msg.ExternalMsgID,
		&msg.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	
	if err != nil {
		slog.Error("Failed to get message by ID",
			"error", err,
			"message_id", id,
		)
		return nil, fmt.Errorf("get message by id: %w", err)
	}
	
	return &msg, nil
}

// Exists checks if a message already exists by external_msg_id (for idempotency)
func (r *MariaDBRepository) Exists(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM messages WHERE external_msg_id = ? LIMIT 1`
	
	var exists int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	
	if err == sql.ErrNoRows {
		return false, nil // Not exists
	}
	
	if err != nil {
		slog.Error("Failed to check message existence",
			"error", err,
			"external_msg_id", id,
		)
		return false, fmt.Errorf("check message existence: %w", err)
	}
	
	return true, nil
}

// ============================================================================
// ConversationRepository Implementation
// ============================================================================

// GetOrCreateByPlatformID retrieves an existing conversation or creates a new one
// Updated for new schema: tenant_id, platform_id, page_id
func (r *MariaDBRepository) GetOrCreateByPlatformID(ctx context.Context, tenantID int, platformID, pageID string) (int64, error) {
	// Try to get existing conversation
	var id int64
	query := `SELECT id FROM conversations WHERE tenant_id = ? AND platform_id = ? AND page_id = ?`
	err := r.db.QueryRowContext(ctx, query, tenantID, platformID, pageID).Scan(&id)
	
	if err == nil {
		// Conversation exists
		return id, nil
	}
	
	if err != sql.ErrNoRows {
		// Unexpected error
		slog.Error("Failed to query conversation",
			"error", err,
			"tenant_id", tenantID,
			"platform_id", platformID,
		)
		return 0, fmt.Errorf("query conversation: %w", err)
	}
	
	// Create new conversation with default values
	// Use empty JSON array for tags
	tagsJSON, _ := json.Marshal([]string{})
	
	insertQuery := `
		INSERT INTO conversations (
			tenant_id, platform_id, page_id, tags, status, created_at
		)
		VALUES (?, ?, ?, ?, ?, NOW())
	`
	
	result, err := r.db.ExecContext(ctx, insertQuery, 
		tenantID, 
		platformID, 
		pageID,
		tagsJSON,
		domain.ConversationStatusUnread,
	)
	if err != nil {
		slog.Error("Failed to create conversation",
			"error", err,
			"tenant_id", tenantID,
			"platform_id", platformID,
		)
		return 0, fmt.Errorf("create conversation: %w", err)
	}
	
	id, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}
	
	slog.Info("New conversation created",
		"conversation_id", id,
		"tenant_id", tenantID,
		"platform_id", platformID,
	)
	
	return id, nil
}
