// Package handler implements HTTP request handlers for the dashboard
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"immortal-chat/internal/adapters/gateway"
	"immortal-chat/internal/adapters/repository"
	"immortal-chat/internal/core/domain"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// DashboardHandler handles dashboard API requests
type DashboardHandler struct {
	db    *sql.DB
	redis *redis.Client
}

// NewDashboardHandler creates a new dashboard handler instance
func NewDashboardHandler(db *sql.DB, rdb *redis.Client) *DashboardHandler {
	return &DashboardHandler{
		db:    db,
		redis: rdb,
	}
}

// ============================================================================
// System Health & Metrics
// ============================================================================

// SystemMetricsResponse represents system health data
type SystemMetricsResponse struct {
	CPUPercent         float64 `json:"cpu_percent"`
	RAMUsedGB          float64 `json:"ram_used_gb"`
	RAMTotalGB         float64 `json:"ram_total_gb"`
	RAMPercent         float64 `json:"ram_percent"`
	DiskUsedGB         float64 `json:"disk_used_gb"`
	DiskTotalGB        float64 `json:"disk_total_gb"`
	DiskPercent        float64 `json:"disk_percent"`
	GoroutinesCount    int     `json:"goroutines_count"`
	WatchdogActive     bool    `json:"watchdog_active"`
	WatchdogThreshold  float64 `json:"watchdog_threshold"`
	DiskWarningLevel   string  `json:"disk_warning_level"` // "safe" | "warning" | "critical"
}

// GetSystemMetrics returns current system health metrics
// GET /api/system/metrics
func (h *DashboardHandler) GetSystemMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// CPU usage (average over 1 second)
	cpuPercents, err := cpu.PercentWithContext(ctx, time.Second, false)
	var cpuPercent float64
	if err == nil && len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}
	
	// Memory stats
	memStat, err := mem.VirtualMemoryWithContext(ctx)
	var ramUsedGB, ramTotalGB, ramPercent float64
	if err == nil {
		ramUsedGB = float64(memStat.Used) / 1024 / 1024 / 1024
		ramTotalGB = float64(memStat.Total) / 1024 / 1024 / 1024
		ramPercent = memStat.UsedPercent
	}
	
	// Disk stats (root partition)
	diskStat, err := disk.UsageWithContext(ctx, ".")
	var diskUsedGB, diskTotalGB, diskPercent float64
	if err == nil {
		diskUsedGB = float64(diskStat.Used) / 1024 / 1024 / 1024
		diskTotalGB = float64(diskStat.Total) / 1024 / 1024 / 1024
		diskPercent = diskStat.UsedPercent
	}
	
	// Goroutines count
	goroutinesCount := runtime.NumGoroutine()
	
	// Watchdog logic (per .rulesgemini)
	watchdogThreshold := 70.0
	watchdogActive := diskPercent > watchdogThreshold
	
	// Determine disk warning level
	var diskWarningLevel string
	switch {
	case diskPercent < 70:
		diskWarningLevel = "safe"
	case diskPercent >= 70 && diskPercent < 80:
		diskWarningLevel = "warning"
	case diskPercent >= 80:
		diskWarningLevel = "critical"
	}
	
	response := SystemMetricsResponse{
		CPUPercent:        roundTo2Decimals(cpuPercent),
		RAMUsedGB:         roundTo2Decimals(ramUsedGB),
		RAMTotalGB:        roundTo2Decimals(ramTotalGB),
		RAMPercent:        roundTo2Decimals(ramPercent),
		DiskUsedGB:        roundTo2Decimals(diskUsedGB),
		DiskTotalGB:       roundTo2Decimals(diskTotalGB),
		DiskPercent:       roundTo2Decimals(diskPercent),
		GoroutinesCount:   goroutinesCount,
		WatchdogActive:    watchdogActive,
		WatchdogThreshold: watchdogThreshold,
		DiskWarningLevel:  diskWarningLevel,
	}
	
	slog.Debug("System metrics retrieved",
		"cpu", cpuPercent,
		"disk_percent", diskPercent,
		"watchdog_active", watchdogActive,
	)
	
	writeJSON(w, http.StatusOK, response)
}

// ============================================================================
// System Status
// ============================================================================

// SystemStatusResponse represents overall system status
type SystemStatusResponse struct {
	Online             bool   `json:"online"`
	Uptime             string `json:"uptime"`
	ActiveConnections  int    `json:"active_connections"`
	Version            string `json:"version"`
	TenantID           int    `json:"tenant_id"`
	StaffRole          string `json:"staff_role"`
	DataScope          string `json:"data_scope"`
}

var appStartTime = time.Now()

// GetStatus returns system status
// GET /api/status
func (h *DashboardHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(appStartTime)
	uptimeStr := formatDuration(uptime)
	
	// Count active goroutines as a proxy for connections
	activeConnections := runtime.NumGoroutine()
	
	response := SystemStatusResponse{
		Online:            true,
		Uptime:            uptimeStr,
		ActiveConnections: activeConnections,
		Version:           "2.0.0",
		TenantID:          1, // TODO: Get from auth context
		StaffRole:         "admin",
		DataScope:         "global",
	}
	
	writeJSON(w, http.StatusOK, response)
}

// ============================================================================
// Platform List
// ============================================================================

// PlatformResponse represents a platform's info
type PlatformResponse struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Platform         string    `json:"platform"`
	Status           string    `json:"status"` // "connected" | "warning" | "error" | "offline"
	Icon             string    `json:"icon"`
	LastActivity     time.Time `json:"last_activity"`
	MessageCountToday int      `json:"message_count_today"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	TokenTTLHours    *int      `json:"token_ttl_hours,omitempty"`
	PendingSync      int       `json:"pending_sync"`
}

// GetPlatforms returns list of all platforms
// GET /api/platforms
func (h *DashboardHandler) GetPlatforms(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Query messages for activity
	query := `
		SELECT 
			COUNT(*) as total_today,
			MAX(created_at) as  last_activity
		FROM messages
		WHERE DATE(created_at) = CURDATE()
	`
	
	var totalToday int
	var lastActivity sql.NullTime
	err := h.db.QueryRowContext(ctx, query).Scan(&totalToday, &lastActivity)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("Failed to query message stats", "error", err)
	}
	
	// Count pending sync
	var pendingSync int
	h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE is_synced = FALSE").Scan(&pendingSync)
	
	// Hardcoded platforms for now (TODO: Get from pages table)
	platforms := []PlatformResponse{
		{
			ID:                1,
			Name:              "Facebook Messenger",
			Platform:          "facebook",
			Status:            determineStatus(lastActivity),
			Icon:              "facebook",
			LastActivity:      getTimeOrNow(lastActivity),
			MessageCountToday: totalToday,
			PendingSync:       pendingSync,
		},
		{
			ID:       2,
			Name:     "Zalo",
			Platform: "zalo",
			Status:   "offline",
			Icon:     "zalo",
		},
		{
			ID:       3,
			Name:     "Telegram",
			Platform: "telegram",
			Status:   "offline",
			Icon:     "telegram",
		},
	}
	
	writeJSON(w, http.StatusOK, platforms)
}

// ============================================================================
// Sync Status
// ============================================================================

// SyncStatusResponse represents federated sync health
type SyncStatusResponse struct {
	PendingMessages     int       `json:"pending_messages"`
	PendingWebhooks     int       `json:"pending_webhooks"`
	LastSyncAt          time.Time `json:"last_sync_at"`
	SyncLagSeconds      int       `json:"sync_lag_seconds"`
	HomeServerReachable bool      `json:"home_server_reachable"`
	SyncHealth          string    `json:"sync_health"` // "healthy" | "lagging" | "critical"
}

// GetSyncStatus returns sync status
// GET /api/sync/status
func (h *DashboardHandler) GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Count pending messages
	var pendingMessages int
	h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE is_synced = FALSE").Scan(&pendingMessages)
	
	// Count pending webhooks
	var pendingWebhooks int
	h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_logs WHERE status = 'pending'").Scan(&pendingWebhooks)
	
	// For now, simulate last sync (TODO: Implement actual federated sync)
	lastSyncAt := time.Now().Add(-2 * time.Minute)
	syncLagSeconds := int(time.Since(lastSyncAt).Seconds())
	
	// Determine sync health
	var syncHealth string
	switch {
	case pendingMessages < 50 && syncLagSeconds < 300:
		syncHealth = "healthy"
	case pendingMessages < 200 && syncLagSeconds < 900:
		syncHealth = "lagging"
	default:
		syncHealth = "critical"
	}
	
	response := SyncStatusResponse{
		PendingMessages:     pendingMessages,
		PendingWebhooks:     pendingWebhooks,
		LastSyncAt:          lastSyncAt,
		SyncLagSeconds:      syncLagSeconds,
		HomeServerReachable: true, // TODO: Implement ping check
		SyncHealth:          syncHealth,
	}
	
	writeJSON(w, http.StatusOK, response)
}

// ============================================================================
// Helpers
// ============================================================================

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func roundTo2Decimals(val float64) float64 {
	return float64(int(val*100)) / 100
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	
	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func determineStatus(lastActivity sql.NullTime) string {
	if !lastActivity.Valid {
		return "offline"
	}
	
	timeSince := time.Since(lastActivity.Time)
	if timeSince < 5*time.Minute {
		return "connected"
	} else if timeSince < 30*time.Minute {
		return "warning"
	}
	
	return "error"
}

func getTimeOrNow(t sql.NullTime) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Now()
}

// ============================================================================
// Phase 3: Conversation Management & Reply APIs
// ============================================================================

// GetConversations returns list of conversations for a specific page
// GET /api/conversations?page_id=xxx
func (h *DashboardHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get page_id from query params
	pageID := r.URL.Query().Get("page_id")
	if pageID == "" {
		// If no page_id provided, use default (first active page)
		// In production, this should come from auth context
		pageID = "770225079500025" // TODO: Get from logged-in user's context
	}
	
	// Call repository
	repo := h.db
	mariadbRepo := repository.NewMariaDBRepository(repo)
	conversations, err := mariadbRepo.GetConversations(ctx, pageID)
	
	if err != nil {
		slog.Error("Failed to get conversations",
			"error", err,
			"page_id", pageID,
		)
		writeJSON(w, http.StatusInternalServerError, InternalErrorResponse("Failed to load conversations"))
		return
	}
	
	// Return with Response Envelope
	writeJSON(w, http.StatusOK, NewSuccessResponse(conversations))
}

// GetConversationMessages returns message history for a conversation
// GET /api/conversations/{id}/messages
// Enhancement: Auto-marks conversation as read when Admin opens chat
func (h *DashboardHandler) GetConversationMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Extract conversation ID from URL path
	// Since we're not using a router, we'll parse manually
	// URL format: /api/conversations/123/messages
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeJSON(w, http.StatusBadRequest, BadRequestResponse("Invalid URL format"))
		return
	}
	
	conversationIDStr := pathParts[3] // /api/conversations/[ID]/messages
	conversationID, err := strconv.ParseInt(conversationIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, BadRequestResponse("Invalid conversation ID"))
		return
	}
	
	// Call repository
	repo := h.db
	mariadbRepo := repository.NewMariaDBRepository(repo)
	messages, err := mariadbRepo.GetMessages(ctx, conversationID)
	
	if err != nil {
		slog.Error("Failed to get messages",
			"error", err,
			"conversation_id", conversationID,
		)
		writeJSON(w, http.StatusInternalServerError, InternalErrorResponse("Failed to load messages"))
		return
	}
	
	// ENHANCEMENT: Auto-mark conversation as read when Admin opens chat
	// This prevents perpetual "unread" badges in UI
	if err := mariadbRepo.MarkConversationAsRead(ctx, conversationID); err != nil {
		// Log but don't fail request (non-critical)
		slog.Warn("Failed to mark conversation as read",
			"error", err,
			"conversation_id", conversationID,
		)
	}
	
	// Return with Response Envelope
	writeJSON(w, http.StatusOK, NewSuccessResponse(messages))
}

// ReplyRequest represents the JSON payload for POST /api/messages/reply
type ReplyRequest struct {
	ConversationID int64  `json:"conversation_id"`
	Text           string `json:"text"`
}

// SendReply handles admin replies to customers via Facebook
// POST /api/messages/reply
// Body: {"conversation_id": 123, "text": "Hello!"}
// 
// ENHANCEMENTS:
// - Auto-deactivates page on token expiry (ErrTokenExpired)
// - User-friendly error messages (not technical details)
// - Proper error recovery per "Core hệ thống lỗi"
func (h *DashboardHandler) SendReply(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Parse request body
	var req ReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, BadRequestResponse("Dữ liệu không hợp lệ"))
		return
	}
	
	// Validate input
	if req.ConversationID == 0 {
		writeJSON(w, http.StatusBadRequest, BadRequestResponse("Thiếu ID hội thoại"))
		return
	}
	
	if strings.TrimSpace(req.Text) == "" {
		writeJSON(w, http.StatusBadRequest, BadRequestResponse("Nội dung tin nhắn không được để trống"))
		return
	}
	
	// Step 1: Get conversation details to find page_id and platform_id
	mariadbRepo := repository.NewMariaDBRepository(h.db)
	
	var platformID, pageID string
	query := `SELECT platform_id, page_id FROM conversations WHERE id = ?`
	err := h.db.QueryRowContext(ctx, query, req.ConversationID).Scan(&platformID, &pageID)
	
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, NotFoundResponse("Không tìm thấy hội thoại"))
		return
	}
	
	if err != nil {
		slog.Error("Failed to get conversation details",
			"error", err,
			"conversation_id", req.ConversationID,
		)
		writeJSON(w, http.StatusInternalServerError, InternalErrorResponse("Lỗi hệ thống khi tra cứu hội thoại"))
		return
	}
	
	// Step 2: Get page access token from database
	accessToken, err := mariadbRepo.GetPageAccessToken(ctx, pageID)
	if err != nil {
		slog.Error("Failed to get page access token",
			"error", err,
			"page_id", pageID,
		)
		writeJSON(w, http.StatusInternalServerError, InternalErrorResponse("Lỗi cấu hình Fanpage. Vui lòng liên hệ quản trị viên"))
		return
	}
	
	// Step 3: Send message via Facebook API
	fbClient := gateway.NewFacebookClient()
	err = fbClient.SendReply(platformID, accessToken, req.Text)
	
	if err != nil {
		// CRITICAL: Handle token death per "Core hệ thống lỗi"
		if errors.Is(err, gateway.ErrTokenExpired) {
			// Auto-deactivate page to prevent futile retries
			if deactivateErr := mariadbRepo.DeactivatePage(ctx, pageID); deactivateErr != nil {
				slog.Error("Failed to deactivate page after token expiry",
					"error", deactivateErr,
					"page_id", pageID,
				)
			}
			
			slog.Warn("🔴 PAGE AUTO-DEACTIVATED",
				"page_id", pageID,
				"conversation_id", req.ConversationID,
				"reason", "Token expired",
			)
			
			// Return user-friendly message
			writeJSON(w, http.StatusBadRequest, BadRequestResponse(
				"Fanpage đã mất kết nối với Facebook. Vui lòng kết nối lại trong phần Cài đặt",
			))
			return
		}
		
		// Handle rate limiting
		if errors.Is(err, gateway.ErrRateLimited) {
			writeJSON(w, http.StatusTooManyRequests, APIResponse{
				Code:    429,
				Message: "Bạn đang gửi tin quá nhanh. Vui lòng chờ vài giây rồi thử lại",
				Data:    nil,
			})
			return
		}
		
		// Handle permission errors
		if errors.Is(err, gateway.ErrPermissionDenied) {
			writeJSON(w, http.StatusForbidden, APIResponse{
				Code:    403,
				Message: "Fanpage không có quyền gửi tin nhắn. Vui lòng kiểm tra cài đặt Facebook",
				Data:    nil,
			})
			return
		}
		
		// Generic Facebook error (network, timeout, etc.)
		slog.Error("Failed to send message via Facebook",
			"error", err,
			"conversation_id", req.ConversationID,
		)
		writeJSON(w, http.StatusInternalServerError, InternalErrorResponse(
			"Không thể gửi tin nhắn. Vui lòng thử lại sau",
		))
		return
	}
	
	// Step 4: Save outbound message to database
	// TODO: Get staff_id from JWT context instead of hardcoding
	outboundMsg := &domain.Message{
		ConversationID: req.ConversationID,
		SenderID:       ptr("admin"), // TODO: Replace with actual staff_id from auth
		SenderType:     domain.SenderTypeAgent,
		Content:        &req.Text,
	}
	
	if err := mariadbRepo.SaveOutboundMessage(ctx, outboundMsg); err != nil {
		// Log error but don't fail the request (message was already sent to Facebook)
		slog.Warn("Failed to save outbound message to DB",
			"error", err,
			"conversation_id", req.ConversationID,
		)
	}
	
	// Step 5: Update conversation's last message
	if err := mariadbRepo.UpdateConversationLastMessage(ctx, req.ConversationID, req.Text); err != nil {
		slog.Warn("Failed to update conversation last message",
			"error", err,
		)
	}
	
	// Return success with user-friendly message
	writeJSON(w, http.StatusOK, NewSuccessResponse(map[string]interface{}{
		"status":          "sent",
		"conversation_id": req.ConversationID,
		"message":         "Tin nhắn đã được gửi thành công",
	}))
}

// Helper to create string pointer
func ptr(s string) *string {
	return &s
}
