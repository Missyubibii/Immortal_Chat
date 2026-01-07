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

// ============================================================================
// Phase 3: Conversation Management & Reply System
// ============================================================================

// ConversationWithSnippet represents a conversation with latest message preview
type ConversationWithSnippet struct {
	ID                 int64  `json:"id"`
	TenantID           int    `json:"tenant_id"`
	PlatformID         string `json:"platform_id"`
	PageID             string `json:"page_id"`
	CustomerName       string `json:"customer_name"`
	LastMessageContent string `json:"last_message_content"`
	LastMessageAt      string `json:"last_message_at"`
	Status             string `json:"status"`
}

// GetConversations retrieves list of conversations ordered by last activity
// Joins with messages to get latest message snippet (Phase 3 Dashboard requirement)
func (r *MariaDBRepository) GetConversations(ctx context.Context, pageID string) ([]ConversationWithSnippet, error) {
	query := `
		SELECT 
			c.id,
			c.tenant_id,
			c.platform_id,
			c.page_id,
			COALESCE(c.customer_name, c.platform_id) as customer_name,
			COALESCE(c.last_message_content, '') as last_message_content,
			COALESCE(c.last_message_at, c.created_at) as last_message_at,
			c.status
		FROM conversations c
		WHERE c.page_id = ?
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC
		LIMIT 100
	`
	
	rows, err := r.db.QueryContext(ctx, query, pageID)
	if err != nil {
		slog.Error("Failed to get conversations",
			"error", err,
			"page_id", pageID,
		)
		return nil, fmt.Errorf("get conversations: %w", err)
	}
	defer rows.Close()
	
	var conversations []ConversationWithSnippet
	for rows.Next() {
		var conv ConversationWithSnippet
		err := rows.Scan(
			&conv.ID,
			&conv.TenantID,
			&conv.PlatformID,
			&conv.PageID,
			&conv.CustomerName,
			&conv.LastMessageContent,
			&conv.LastMessageAt,
			&conv.Status,
		)
		if err != nil {
			slog.Error("Failed to scan conversation row", "error", err)
			continue
		}
		conversations = append(conversations, conv)
	}
	
	slog.Info("Retrieved conversations",
		"page_id", pageID,
		"count", len(conversations),
	)
	
	return conversations, nil
}

// GetMessages retrieves all messages for a specific conversation
// Ordered by created_at ASC (oldest first) for chat display (Phase 3)
func (r *MariaDBRepository) GetMessages(ctx context.Context, conversationID int64) ([]*domain.Message, error) {
	query := `
		SELECT 
			id, conversation_id, sender_id, sender_type, content,
			attachments, type, is_synced, external_msg_id, created_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
		LIMIT 1000
	`
	
	rows, err := r.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		slog.Error("Failed to get messages",
			"error", err,
			"conversation_id", conversationID,
		)
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()
	
	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
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
		if err != nil {
			slog.Error("Failed to scan message row", "error", err)
			continue
		}
		messages = append(messages, &msg)
	}
	
	slog.Info("Retrieved messages",
		"conversation_id", conversationID,
		"count", len(messages),
	)
	
	return messages, nil
}

// SaveOutboundMessage persists a reply message sent by Admin to customer
// Used after successful Facebook Send API call (Phase 3)
func (r *MariaDBRepository) SaveOutboundMessage(ctx context.Context, msg *domain.Message) error {
	query := `
		INSERT INTO messages (
			conversation_id, sender_id, sender_type, content,
			attachments, type, is_synced, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
	`
	
	// For outbound messages, we use empty attachments and set type to 'text'
	emptyAttachments := json.RawMessage("[]")
	textType := "text"
	
	_, err := r.db.ExecContext(ctx, query,
		msg.ConversationID,
		msg.SenderID,      // Admin ID or "system"
		domain.SenderTypeAgent, // Always 'agent' for admin replies
		msg.Content,
		emptyAttachments,
		textType,
		false, // is_synced = false initially
	)
	
	if err != nil {
		slog.Error("Failed to save outbound message",
			"error", err,
			"conversation_id", msg.ConversationID,
		)
		return fmt.Errorf("save outbound message: %w", err)
	}
	
	slog.Info("Outbound message saved",
		"conversation_id", msg.ConversationID,
		"sender_type", "agent",
	)
	
	return nil
}

// GetPageAccessToken retrieves the access token for a Facebook page
// Required for Send API calls (Phase 3)
func (r *MariaDBRepository) GetPageAccessToken(ctx context.Context, pageID string) (string, error) {
	query := `SELECT access_token FROM pages WHERE page_id = ? AND is_active = TRUE LIMIT 1`
	
	var accessToken string
	err := r.db.QueryRowContext(ctx, query, pageID).Scan(&accessToken)
	
	if err == sql.ErrNoRows {
		slog.Warn("No active page found", "page_id", pageID)
		return "", fmt.Errorf("page not found or inactive")
	}
	
	if err != nil {
		slog.Error("Failed to get page access token",
			"error", err,
			"page_id", pageID,
		)
		return "", fmt.Errorf("get page access token: %w", err)
	}
	
	slog.Debug("Retrieved page access token", "page_id", pageID)
	return accessToken, nil
}

// UpdateConversationLastMessage updates conversation metadata after new message
// Used to keep conversation list fresh (Phase 3)
func (r *MariaDBRepository) UpdateConversationLastMessage(ctx context.Context, conversationID int64, content string) error {
	query := `
		UPDATE conversations
		SET last_message_content = ?,
			last_message_at = NOW(),
			updated_at = NOW()
		WHERE id = ?
	`
	
	_, err := r.db.ExecContext(ctx, query, content, conversationID)
	if err != nil {
		slog.Error("Failed to update conversation last message",
			"error", err,
			"conversation_id", conversationID,
		)
		return fmt.Errorf("update conversation: %w", err)
	}
	
	return nil
}

// ============================================================================
// Phase 3 Enhancement: Resilience & Error Recovery
// ============================================================================

// DeactivatePage disables a page when token expires or becomes invalid
// Per "Core hệ thống lỗi": AUTO deactivate to prevent futile API calls
func (r *MariaDBRepository) DeactivatePage(ctx context.Context, pageID string) error {
	query := `
		UPDATE pages
		SET is_active = FALSE
		WHERE page_id = ?
	`
	
	result, err := r.db.ExecContext(ctx, query, pageID)
	if err != nil {
		slog.Error("Failed to deactivate page",
			"error", err,
			"page_id", pageID,
		)
		return fmt.Errorf("deactivate page: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Warn("🔴 PAGE DEACTIVATED - Token expired or invalid",
			"page_id", pageID,
			"action", "Admin must reconnect Facebook",
		)
	}
	
	return nil
}

// MarkConversationAsRead updates conversation status to 'read'
// Called when Admin opens chat window to prevent perpetual "unread" badge
func (r *MariaDBRepository) MarkConversationAsRead(ctx context.Context, conversationID int64) error {
	query := `
		UPDATE conversations
		SET status = 'read',
			updated_at = NOW()
		WHERE id = ? AND status = 'unread'
	`
	
	result, err := r.db.ExecContext(ctx, query, conversationID)
	if err != nil {
		slog.Error("Failed to mark conversation as read",
			"error", err,
			"conversation_id", conversationID,
		)
		return fmt.Errorf("mark as read: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows > 0 {
		slog.Debug("Conversation marked as read",
			"conversation_id", conversationID,
		)
	}
	
	return nil
}

// GetStaffInfo retrieves staff information for message metadata
// Returns staff_id and name for audit trail (not hardcoded "admin")
func (r *MariaDBRepository) GetStaffInfo(ctx context.Context, staffID int) (string, string, error) {
	// TODO: Implement after staff table is created
	// For now, return default
	return "1", "Admin", nil
}
