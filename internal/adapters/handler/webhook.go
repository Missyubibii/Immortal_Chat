// Package handler implements HTTP request handlers
// Following Hexagonal Architecture: Adapters translate HTTP to domain logic
package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"immortal-chat/internal/core/services"
)

// WebhookHandler handles Facebook webhook verification and events
// Per .rulesgemini Section 4: Must respond < 3 seconds
type WebhookHandler struct {
	dispatcher  *services.Dispatcher
	appSecret   string // For HMAC signature validation
	verifyToken string // For webhook verification
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(dispatcher *services.Dispatcher, appSecret, verifyToken string) *WebhookHandler {
	return &WebhookHandler{
		dispatcher:  dispatcher,
		appSecret:   appSecret,
		verifyToken: verifyToken,
	}
}

// ============================================================================
// GET /webhook/facebook - Webhook Verification
// ============================================================================

// HandleFacebookVerify handles webhook verification challenge from Facebook
// Ref: https://developers.facebook.com/docs/messenger-platform/webhooks#verification
func (h *WebhookHandler) HandleFacebookVerify(w http.ResponseWriter, r *http.Request) {
	// Facebook sends these query parameters:
	// - hub.mode: "subscribe"
	// - hub.verify_token: your custom token
	// - hub.challenge: random string to echo back
	
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	slog.Info("Webhook verification request received",
		"mode", mode,
		"token_matches", token == h.verifyToken,
	)

	// Verify the token matches
	if mode == "subscribe" && token == h.verifyToken {
		// Verification successful - return the challenge
		slog.Info("Webhook verification successful")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	// Verification failed
	slog.Warn("Webhook verification failed",
		"expected_token", h.verifyToken,
		"received_token", token,
		"mode", mode,
	)
	
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// ============================================================================
// POST /webhook/facebook - Webhook Events
// ============================================================================

// HandleFacebookEvent handles incoming Facebook webhook events
// Per .rulesgemini Section 4: Fire & Forget - return 200 OK immediately
// Per user requirement: Validate HMAC signature before processing
func (h *WebhookHandler) HandleFacebookEvent(w http.ResponseWriter, r *http.Request) {
	// ========================================================================
	// Step 1: Read request body
	// ========================================================================
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read webhook body", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// ========================================================================
	// Step 2: CRITICAL - Validate HMAC Signature
	// Per .rulesgemini Section 4: Do NOT process without valid signature
	// Per user requirement: Security is mandatory
	// ========================================================================
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		slog.Warn("Webhook received without signature header")
		http.Error(w, "Forbidden - No signature", http.StatusForbidden)
		return
	}

	if !h.validateSignature(body, signature) {
		slog.Warn("Webhook signature validation failed",
			"signature", signature,
		)
		http.Error(w, "Forbidden - Invalid signature", http.StatusForbidden)
		return
	}

	slog.Debug("Webhook signature validated successfully")

	// ========================================================================
	// Step 3: Return HTTP 200 OK IMMEDIATELY (Fire & Forget)
	// Per .rulesgemini Section 4: Must respond < 3 seconds
	// ========================================================================
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("EVENT_RECEIVED"))

	// ========================================================================
	// Step 4: Process webhook asynchronously in goroutine
	// Per user requirement: Use defer recover() to prevent container crash
	// ========================================================================
	go func() {
		// Panic recovery per user requirement
		defer func() {
			if r := recover(); r != nil {
				slog.Error("PANIC in webhook processing goroutine",
					"panic", r,
				)
			}
		}()

		// Process webhook with dispatcher
		// Platform is hardcoded as "facebook" for this handler
		h.dispatcher.ProcessWebhook(r.Context(), "facebook", body)
	}()

	slog.Info("Webhook received and queued for processing",
		"content_length", len(body),
	)
}

// ============================================================================
// HMAC Signature Validation
// ============================================================================

// validateSignature validates the HMAC SHA256 signature from Facebook
// Per .rulesgemini: Security requirement - reject invalid signatures
// Ref: https://developers.facebook.com/docs/messenger-platform/webhooks#security
func (h *WebhookHandler) validateSignature(payload []byte, signatureHeader string) bool {
	// Facebook sends signature in format: "sha256=<hex_signature>"
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		slog.Warn("Invalid signature format - missing sha256= prefix")
		return false
	}

	// Extract hex signature
	expectedSignature := strings.TrimPrefix(signatureHeader, prefix)

	// Compute HMAC SHA256
	mac := hmac.New(sha256.New, []byte(h.appSecret))
	mac.Write(payload)
	computedSignatureBytes := mac.Sum(nil)
	computedSignature := hex.EncodeToString(computedSignatureBytes)

	// Compare signatures (constant-time comparison to prevent timing attacks)
	isValid := hmac.Equal(
		[]byte(computedSignature),
		[]byte(expectedSignature),
	)

	if !isValid {
		slog.Warn("HMAC signature mismatch",
			"expected", expectedSignature,
			"computed", computedSignature,
		)
	}

	return isValid
}

// ============================================================================
// Response Helper (Per .rulesgemini Section 7: Standard JSON envelope)
// ============================================================================

// JSONResponse represents the standard API response format
// Per .rulesgemini Section 7: All APIs must use this envelope
type JSONResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

// WriteJSON writes a JSON response (helper for future endpoints)
func WriteJSON(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := JSONResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	
	// In production, add trace_id for request tracking
	fmt.Fprintf(w, `{"code":%d,"message":"%s"}`, response.Code, response.Message)
}
