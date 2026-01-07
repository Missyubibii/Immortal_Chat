// Package dto contains data transfer objects for external APIs
// Separating DTOs from handlers prevents import cycles
package dto

// FacebookWebhookRequest is the top-level webhook payload from Facebook
// Ref: https://developers.facebook.com/docs/messenger-platform/webhooks
type FacebookWebhookRequest struct {
	Object string          `json:"object"` // Always "page" for Messenger
	Entry  []FacebookEntry `json:"entry"`  // Array of page entries
}

// FacebookEntry represents a single page's webhook events
type FacebookEntry struct {
	ID        string               `json:"id"`        // Page ID
	Time      int64                `json:"time"`      // Event timestamp (Unix milliseconds)
	Messaging []FacebookMessaging  `json:"messaging"` // Array of messaging events
}

// FacebookMessaging represents a single messaging event
// Can be a message, delivery receipt, read receipt, or echo
type FacebookMessaging struct {
	Sender    FacebookUser     `json:"sender"`              // Who sent the message
	Recipient FacebookUser     `json:"recipient"`           // Who received the message
	Timestamp int64            `json:"timestamp"`           // Event timestamp (Unix milliseconds)
	
	// Message event (user sent a message to the page)
	Message   *FacebookMessage `json:"message,omitempty"`   // Actual message content
	
	// Echo event (page sent a message to user - NOT a user message)
	// Per user requirement: We MUST filter these out
	
	// Delivery confirmation (message was delivered)
	Delivery  *FacebookDelivery `json:"delivery,omitempty"`  // Delivery receipt
	
	// Read confirmation (message was read)
	Read      *FacebookRead     `json:"read,omitempty"`      // Read receipt
}

// FacebookUser represents a sender or recipient (PSID)
type FacebookUser struct {
	ID string `json:"id"` // Page-Scoped ID (PSID)
}

// FacebookMessage represents the actual message content
type FacebookMessage struct {
	MID  string `json:"mid"`  // Message ID (used for deduplication)
	Text string `json:"text"` // Text content
	
	// Attachments (images, files, etc.)
	Attachments []FacebookAttachment `json:"attachments,omitempty"`
	
	// IsEcho indicates this message was sent BY the page (not TO the page)
	// CRITICAL: We must filter these out per user requirement
	IsEcho bool `json:"is_echo,omitempty"`
}

// FacebookAttachment represents media attachments
type FacebookAttachment struct {
	Type    string                `json:"type"`    // "image", "video", "audio", "file"
	Payload FacebookAttachmentPayload `json:"payload"` // Attachment details
}

// FacebookAttachmentPayload contains attachment URL and metadata
type FacebookAttachmentPayload struct {
	URL string `json:"url"` // Download URL for the attachment
}

// FacebookDelivery represents a delivery confirmation
type FacebookDelivery struct {
	MIDs      []string `json:"mids"`      // Array of delivered message IDs
	Watermark int64    `json:"watermark"` // Timestamp
}

// FacebookRead represents a read confirmation
type FacebookRead struct {
	Watermark int64 `json:"watermark"` // All messages before this timestamp were read
}

// IsUserMessage determines if this messaging event is an actual user message
// Returns false for: echo messages, delivery receipts, read receipts
// Per user requirement: Only save real user messages to database
func (m *FacebookMessaging) IsUserMessage() bool {
	// If Message field is nil, this is not a message event
	if m.Message == nil {
		return false
	}
	
	// Filter out echo messages (sent by the page itself)
	if m.Message.IsEcho {
		return false
	}
	
	// Filter out delivery receipts
	if m.Delivery != nil {
		return false
	}
	
	// Filter out read receipts
	if m.Read != nil {
		return false
	}
	
	// This is a genuine user message
	return true
}

// GetMessageID extracts the message ID for deduplication
func (m *FacebookMessaging) GetMessageID() string {
	if m.Message != nil {
		return m.Message.MID
	}
	return ""
}

// GetMessageType determines the type of message
func (m *FacebookMessaging) GetMessageType() string {
	if m.Message == nil {
		return ""
	}
	
	// Check for attachments first
	if len(m.Message.Attachments) > 0 {
		return m.Message.Attachments[0].Type // "image", "video", "audio", "file"
	}
	
	// Default to text if no attachments
	if m.Message.Text != "" {
		return "text"
	}
	
	return "unknown"
}

// GetContent extracts the message content (text or attachment URL)
func (m *FacebookMessaging) GetContent() string {
	if m.Message == nil {
		return ""
	}
	
	// Text messages
	if m.Message.Text != "" {
		return m.Message.Text
	}
	
	// Attachment URLs
	if len(m.Message.Attachments) > 0 && m.Message.Attachments[0].Payload.URL != "" {
		return m.Message.Attachments[0].Payload.URL
	}
	
	return ""
}
