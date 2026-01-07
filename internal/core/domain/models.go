// Package domain contains core business entities
// Following Hexagonal Architecture: These models are infrastructure-agnostic
package domain

import (
	"encoding/json"
	"time"
)

// WebhookLog represents the audit trail for incoming webhook events
// Updated to match new schema from technical specification document
type WebhookLog struct {
	ID          int64           `json:"id" db:"id"`
	Platform    string          `json:"platform" db:"platform"`                 // "facebook", "zalo"
	PayloadJSON json.RawMessage `json:"payload_json" db:"payload_json"`         // JSON field
	Status      string          `json:"status" db:"status"`                     // "pending", "processed", "failed"
	RetryCount  int             `json:"retry_count" db:"retry_count"`           // Number of retry attempts
	ErrorLog    *string         `json:"error_log,omitempty" db:"error_log"`     // Error details if failed
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
}

// WebhookStatus constants for lifecycle management
const (
	WebhookStatusPending   = "pending"
	WebhookStatusProcessed = "processed"
	WebhookStatusFailed    = "failed"
)

// Message represents a parsed chat message from any platform
// Updated to match new schema from technical specification document
type Message struct {
	ID             int64           `json:"id" db:"id"`
	ConversationID int64           `json:"conversation_id" db:"conversation_id"`
	SenderID       *string         `json:"sender_id,omitempty" db:"sender_id"`
	SenderType     string          `json:"sender_type" db:"sender_type"`           // "user", "bot", "agent"
	Content        *string         `json:"content,omitempty" db:"content"`
	Attachments    json.RawMessage `json:"attachments,omitempty" db:"attachments"` // JSON field
	Type           *string         `json:"type,omitempty" db:"type"`               // "text", "image", "file", "sticker", "voice"
	IsSynced       bool            `json:"is_synced" db:"is_synced"`
	ExternalMsgID  *string         `json:"external_msg_id,omitempty" db:"external_msg_id"` // Platform message ID (for dedup)
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// SenderType constants
const (
	SenderTypeUser  = "user"
	SenderTypeBot   = "bot"
	SenderTypeAgent = "agent"
)

// MessageType constants
const (
	MessageTypeText    = "text"
	MessageTypeImage   = "image"
	MessageTypeFile    = "file"
	MessageTypeSticker = "sticker"
	MessageTypeVoice   = "voice"
)

// Conversation represents a chat thread/conversation
// Updated to match new schema from technical specification document
type Conversation struct {
	ID                 int64           `json:"id" db:"id"`
	TenantID           int             `json:"tenant_id" db:"tenant_id"`
	PlatformID         string          `json:"platform_id" db:"platform_id"`     // Platform-specific conversation ID
	PageID             *string         `json:"page_id,omitempty" db:"page_id"`
	CustomerName       *string         `json:"customer_name,omitempty" db:"customer_name"`
	LastMessageContent *string         `json:"last_message_content,omitempty" db:"last_message_content"`
	LastMessageAt      *time.Time      `json:"last_message_at,omitempty" db:"last_message_at"`
	Tags               json.RawMessage `json:"tags,omitempty" db:"tags"`         // JSON field
	AssigneeID         *int            `json:"assignee_id,omitempty" db:"assignee_id"`
	Status             string          `json:"status" db:"status"`               // "unread", "read", "archived"
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
}

// ConversationStatus constants
const (
	ConversationStatusUnread   = "unread"
	ConversationStatusRead     = "read"
	ConversationStatusArchived = "archived"
)

// Tenant represents a customer/tenant in the multi-tenant system
type Tenant struct {
	ID        int             `json:"id" db:"id"`
	Name      string          `json:"name" db:"name"`
	Plan      string          `json:"plan" db:"plan"`                       // "basic", "pro", "vip"
	ExpiredAt *time.Time      `json:"expired_at,omitempty" db:"expired_at"`
	Config    json.RawMessage `json:"config,omitempty" db:"config"`         // JSON field
	IsActive  bool            `json:"is_active" db:"is_active"`
}

// Page represents a connected Facebook/Zalo page
type Page struct {
	ID          int64   `json:"id" db:"id"`
	TenantID    int     `json:"tenant_id" db:"tenant_id"`
	Platform    string  `json:"platform" db:"platform"`       // "facebook", "zalo"
	PageID      string  `json:"page_id" db:"page_id"`
	PageName    *string `json:"page_name,omitempty" db:"page_name"`
	AccessToken string  `json:"-" db:"access_token"`          // Never expose in JSON
	IsActive    bool    `json:"is_active" db:"is_active"`
}
