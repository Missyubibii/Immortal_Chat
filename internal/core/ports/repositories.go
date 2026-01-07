// Package ports defines interfaces for dependency inversion
// Following Hexagonal Architecture: Core defines contracts, Adapters implement them
package ports

import (
	"context"
	"time"

	"immortal-chat/internal/core/domain"
)

// WebhookRepository handles persistence of webhook audit logs
// Per .rulesgemini Section 3: All webhooks must be logged for audit and replay
type WebhookRepository interface {
	// SaveLog persists a webhook event to the audit log
	SaveLog(ctx context.Context, log *domain.WebhookLog) error
	
	// UpdateStatus updates the processing status of a webhook log
	// Used to track lifecycle: pending -> processed/failed
	UpdateStatus(ctx context.Context, id string, status string) error
}

// MessageRepository handles persistence of parsed chat messages
// Per .rulesgemini Section 3: Local-First data storage
type MessageRepository interface {
	// SaveMessage persists a parsed message to the database
	SaveMessage(ctx context.Context, msg *domain.Message) error
	
	// GetByID retrieves a message by its platform-specific ID
	GetByID(ctx context.Context, id string) (*domain.Message, error)
	
	// Exists checks if a message with the given ID already exists
	// Used for idempotency checks (alternative to Redis dedup)
	Exists(ctx context.Context, id string) (bool, error)
}

// ConversationRepository handles conversation/thread management
type ConversationRepository interface {
	// GetOrCreateByPlatformID retrieves an existing conversation or creates a new one
	// Uses platform_id (e.g., Facebook PSID) instead of external_id
	// Returns conversation database ID for linking messages
	GetOrCreateByPlatformID(ctx context.Context, tenantID int, platformID, pageID string) (int64, error)
}

// DedupRepository handles deduplication of webhook events using cache
// Per .rulesgemini Section 4: Check dedup before processing
type DedupRepository interface {
	// IsDuplicate checks if an event ID has already been processed
	// Returns true if the event exists in cache (is a duplicate)
	IsDuplicate(ctx context.Context, eventID string) (bool, error)
	
	// MarkProcessed marks an event as processed in the cache
	// Sets a TTL to automatically expire old entries
	MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) error
}
