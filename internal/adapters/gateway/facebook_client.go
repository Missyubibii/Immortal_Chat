// Package gateway implements external API adapters
// Following Hexagonal Architecture: Outbound adapters for external services
package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Custom errors for specific Facebook API failures
var (
	// ErrTokenExpired indicates the page access token is expired or invalid (code 190)
	// Handler should call DeactivatePage() when this error is received
	ErrTokenExpired = errors.New("facebook access token expired or invalid")
	
	// ErrRateLimited indicates Facebook rate limit exceeded (code 4, 17, 32, 613)
	ErrRateLimited = errors.New("facebook rate limit exceeded")
	
	// ErrPermissionDenied indicates missing permissions (code 10, 200, 299)
	ErrPermissionDenied = errors.New("facebook permission denied")
)

// FacebookClient handles communication with Facebook Graph API
// Phase 3: Send messages back to customers
type FacebookClient struct {
	httpClient *http.Client
	apiVersion string
}

// NewFacebookClient creates a new Facebook API client
func NewFacebookClient() *FacebookClient {
	return &FacebookClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiVersion: "v19.0", // Facebook Graph API version
	}
}

// SendMessageRequest represents the Facebook Send API payload structure
type SendMessageRequest struct {
	Recipient struct {
		ID string `json:"id"` // PSID (Page-Scoped ID)
	} `json:"recipient"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	MessagingType string `json:"messaging_type"` // "RESPONSE" for replies
}

// SendMessageResponse represents Facebook's response
type SendMessageResponse struct {
	RecipientID string `json:"recipient_id"`
	MessageID   string `json:"message_id"`
}

// FacebookError represents an error from Facebook API
type FacebookError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode"`
	FBTraceID    string `json:"fbtrace_id"`
}

// SendReply sends a text message to a Facebook user with retry mechanism
// recipientPSID: Page-Scoped User ID (from conversations.platform_id)
// pageAccessToken: From database (pages.access_token)
// text: The message content to send
// 
// Returns specific errors:
// - ErrTokenExpired: Token invalid/expired (code 190) → Caller should deactivate page
// - ErrRateLimited: Rate limit exceeded → Caller should retry later
// - ErrPermissionDenied: Missing permissions
func (c *FacebookClient) SendReply(recipientPSID, pageAccessToken, text string) error {
	const maxRetries = 3
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := c.sendReplyAttempt(recipientPSID, pageAccessToken, text, attempt)
		
		if err == nil {
			return nil // Success
		}
		
		// Don't retry on these specific errors
		if errors.Is(err, ErrTokenExpired) ||
			errors.Is(err, ErrPermissionDenied) ||
			errors.Is(err, ErrRateLimited) {
			return err
		}
		
		// Retry on network errors with exponential backoff
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			slog.Warn("Retrying Facebook API call",
				"attempt", attempt,
				"max_retries", maxRetries,
				"backoff_ms", backoff.Milliseconds(),
				"error", err,
			)
			time.Sleep(backoff)
		}
	}
	
	return fmt.Errorf("failed after %d attempts", maxRetries)
}

// sendReplyAttempt performs a single attempt to send message
func (c *FacebookClient) sendReplyAttempt(recipientPSID, pageAccessToken, text string, attempt int) error {
	// Construct the API URL
	url := fmt.Sprintf("https://graph.facebook.com/%s/me/messages", c.apiVersion)
	
	// Build request payload
	payload := SendMessageRequest{
		MessagingType: "RESPONSE",
	}
	payload.Recipient.ID = recipientPSID
	payload.Message.Text = text
	
	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.URL.RawQuery = fmt.Sprintf("access_token=%s", pageAccessToken)
	
	// Log outgoing request (without token for security)
	slog.Info("Sending message to Facebook",
		"recipient_psid", recipientPSID,
		"text_length", len(text),
		"attempt", attempt,
	)
	
	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send request to Facebook",
			"error", err,
			"attempt", attempt,
		)
		return fmt.Errorf("facebook api request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		// Parse error response
		var fbError struct {
			Error FacebookError `json:"error"`
		}
		
		if err := json.Unmarshal(body, &fbError); err != nil {
			slog.Error("Facebook API error (unparseable)",
				"status_code", resp.StatusCode,
				"body", string(body),
			)
			return fmt.Errorf("facebook api error %d: %s", resp.StatusCode, string(body))
		}
		
		slog.Error("Facebook API error",
			"status_code", resp.StatusCode,
			"error_code", fbError.Error.Code,
			"error_message", fbError.Error.Message,
			"error_subcode", fbError.Error.ErrorSubcode,
			"fbtrace_id", fbError.Error.FBTraceID,
		)
		
		// Return specific errors based on code
		switch fbError.Error.Code {
		case 190: // Token expired/invalid
			return ErrTokenExpired
		case 4, 17, 32, 613: // Rate limiting
			return ErrRateLimited
		case 10, 200, 299: // Permission errors
			return ErrPermissionDenied
		case 100: // Invalid parameter
			return fmt.Errorf("invalid parameter: %s", fbError.Error.Message)
		default:
			return fmt.Errorf("facebook api error (code %d): %s", fbError.Error.Code, fbError.Error.Message)
		}
	}
	
	// Parse success response
	var sendResp SendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		slog.Warn("Failed to parse success response",
			"error", err,
			"body", string(body),
		)
		// Still return nil since HTTP 200 means it worked
		return nil
	}
	
	slog.Info("Message sent successfully",
		"recipient_psid", recipientPSID,
		"message_id", sendResp.MessageID,
		"attempt", attempt,
	)
	
	return nil
}

// SendTypingIndicator sends a typing indicator (optional enhancement)
// Shows "..." bubbles in customer's Messenger
func (c *FacebookClient) SendTypingIndicator(recipientPSID, pageAccessToken string, action string) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/me/messages", c.apiVersion)
	
	payload := map[string]interface{}{
		"recipient": map[string]string{
			"id": recipientPSID,
		},
		"sender_action": action, // "typing_on" or "typing_off"
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.URL.RawQuery = fmt.Sprintf("access_token=%s", pageAccessToken)
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Warn("Failed to send typing indicator",
			"status", resp.StatusCode,
			"body", string(body),
		)
	}
	
	return nil
}
