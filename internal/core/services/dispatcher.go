// Package services contains core business logic
// Following Hexagonal Architecture: Services orchestrate domain logic using ports
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"immortal-chat/internal/adapters/dto"
	"immortal-chat/internal/core/domain"
	"immortal-chat/internal/core/ports"
)

// Dispatcher orchestrates webhook processing workflow
// Per .rulesgemini Section 4: Fire & Forget pattern with async processing
type Dispatcher struct {
	webhookRepo      ports.WebhookRepository
	messageRepo      ports.MessageRepository
	conversationRepo ports.ConversationRepository
	dedupRepo        ports.DedupRepository
}

// NewDispatcher creates a new dispatcher instance with dependencies injected
func NewDispatcher(
	webhookRepo ports.WebhookRepository,
	messageRepo ports.MessageRepository,
	conversationRepo ports.ConversationRepository,
	dedupRepo ports.DedupRepository,
) *Dispatcher {
	return &Dispatcher{
		webhookRepo:      webhookRepo,
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		dedupRepo:        dedupRepo,
	}
}

// ProcessWebhook processes an incoming Facebook webhook payload
// Per user requirement: Filter echo/delivery/read messages, handle panics gracefully
// Per .rulesgemini Section 4: Response must be < 3 seconds
func (d *Dispatcher) ProcessWebhook(ctx context.Context, platform string, payload []byte) {
	// ========================================================================
	// CRITICAL: Panic Recovery per user requirement
	// Prevents Docker container crash when processing fails
	// ========================================================================
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC recovered in ProcessWebhook",
				"panic", r,
				"platform", platform,
			)
			// Log panic but don't crash the application
		}
	}()

	// ========================================================================
	// Step 1: Save webhook to audit log (Async pattern)
	// Per .rulesgemini Section 3: All webhook data must be persisted
	// ========================================================================
	webhookLog := &domain.WebhookLog{
		Platform:    platform,
		PayloadJSON: json.RawMessage(payload),
		Status:      domain.WebhookStatusPending,
		RetryCount:  0,
		CreatedAt:   time.Now(),
	}

	// Save webhook log (don't block on errors)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("PANIC in webhook log save", "panic", r)
			}
		}()
		
		if err := d.webhookRepo.SaveLog(context.Background(), webhookLog); err != nil {
			slog.Error("Failed to save webhook log (async)",
				"error", err,
			)
		}
	}()

	// ========================================================================
	// Step 2: Parse Facebook webhook payload
	// ========================================================================
	var fbPayload dto.FacebookWebhookRequest
	if err := json.Unmarshal(payload, &fbPayload); err != nil {
		slog.Error("Failed to parse Facebook webhook JSON",
			"error", err,
		)
		// Note: Can't update webhook status without ID from insert
		return
	}

	// ========================================================================
	// Step 3: Process each messaging event in the webhook
	// Facebook can send multiple events in one webhook call
	// Use background context for async processing (request context will be cancelled)
	// ========================================================================
	processedCount := 0
	skippedCount := 0
	
	// Create background context for async processing
	bgCtx := context.Background()

	for _, entry := range fbPayload.Entry {
		for _, messaging := range entry.Messaging {
			// ================================================================
			// CRITICAL: Filter non-user messages per user requirement
			// Do NOT save echo messages, delivery receipts, or read receipts
			// ================================================================
			if !messaging.IsUserMessage() {
				slog.Debug("Skipping non-user message event",
					"is_echo", messaging.Message != nil && messaging.Message.IsEcho,
					"has_delivery", messaging.Delivery != nil,
					"has_read", messaging.Read != nil,
				)
				skippedCount++
				continue
			}

			// Process the user message with background context
			if err := d.processMessage(bgCtx, platform, &messaging); err != nil {
				slog.Error("Failed to process message",
					"error", err,
					"message_id", messaging.GetMessageID(),
				)
				// Continue processing other messages even if one fails
			} else {
				processedCount++
			}
		}
	}

	slog.Info("Webhook processing completed",
		"processed", processedCount,
		"skipped", skippedCount,
	)
}

// processMessage handles a single messaging event
func (d *Dispatcher) processMessage(ctx context.Context, platform string, messaging *dto.FacebookMessaging) error {
	messageID := messaging.GetMessageID()
	
	// ========================================================================
	// Step 1: Check for duplicates
	// Per .rulesgemini Section 4: Deduplication check
	// ========================================================================
	isDup, err := d.dedupRepo.IsDuplicate(ctx, messageID)
	if err != nil {
		return fmt.Errorf("dedup check failed: %w", err)
	}
	
	if isDup {
		slog.Info("Duplicate message detected, skipping",
			"message_id", messageID,
		)
		return nil // Not an error, just skip
	}

	// ========================================================================
	// Step 2: Get or create conversation
	// Use tenant_id = 1 as default (for single-tenant setup)
	// In multi-tenant, this would come from page configuration
	// ========================================================================
	tenantID := 1 // TODO: Get from page/tenant mapping
	platformID := messaging.Sender.ID // Facebook PSID
	pageID := messaging.Recipient.ID  // Facebook Page ID
	
	conversationID, err := d.conversationRepo.GetOrCreateByPlatformID(ctx, tenantID, platformID, pageID)
	if err != nil {
		return fmt.Errorf("get/create conversation failed: %w", err)
	}

	// ========================================================================
	// Step 3: Build domain message entity
	// ========================================================================
	content := messaging.GetContent()
	msgType := messaging.GetMessageType()
	
	// Create empty JSON array for attachments
	attachmentsJSON := json.RawMessage("[]")
	if len(messaging.Message.Attachments) > 0 {
		attachmentsJSON, _ = json.Marshal(messaging.Message.Attachments)
	}
	
	message := &domain.Message{
		ConversationID: conversationID,
		SenderID:       &messaging.Sender.ID,
		SenderType:     domain.SenderTypeUser, // Always user for incoming messages
		Content:        &content,
		Attachments:    attachmentsJSON,
		Type:           &msgType,
		IsSynced:       false,
		ExternalMsgID:  &messageID,
		CreatedAt:      time.Now(),
	}

	// ========================================================================
	// Step 4: Save message to database
	// Per .rulesgemini Section 3: Local-First data storage
	// ========================================================================
	if err := d.messageRepo.SaveMessage(ctx, message); err != nil {
		return fmt.Errorf("save message failed: %w", err)
	}

	// ========================================================================
	// Step 5: Mark as processed in dedup cache
	// Per .rulesgemini: TTL 10 minutes minimum, we use 24 hours for safety
	// ========================================================================
	if err := d.dedupRepo.MarkProcessed(ctx, messageID, 24*time.Hour); err != nil {
		// Log error but don't fail (message already saved)
		slog.Warn("Failed to mark message in dedup cache",
			"error", err,
			"message_id", messageID,
		)
	}

	contentPreview := content
	if len(contentPreview) > 50 {
		contentPreview = contentPreview[:50] + "..."
	}
	
	slog.Info("Message processed successfully",
		"message_id", messageID,
		"conversation_id", conversationID,
		"sender_id", messaging.Sender.ID,
		"content_preview", contentPreview,
	)

	return nil
}

// updateWebhookStatus updates webhook log status (fire and forget)
// Note: Currently disabled because we don't have webhook ID after async insert
func (d *Dispatcher) updateWebhookStatus(webhookID string, status string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("PANIC in webhook status update", "panic", r, "webhook_id", webhookID)
			}
		}()
		
		ctx := context.Background()
		if err := d.webhookRepo.UpdateStatus(ctx, webhookID, status); err != nil {
			slog.Error("Failed to update webhook status",
				"error", err,
				"webhook_id", webhookID,
				"status", status,
			)
		}
	}()
}

// Helper to convert string to int64
func toInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
