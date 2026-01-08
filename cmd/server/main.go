// Package main - Immortal Chat OS Application Entry Point
// Merged Phase 2 & Phase 3: Dashboard, Chat, Monitoring & Resilience
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	// Bổ sung Gateway cho Facebook
	"immortal-chat/internal/adapters/handler"
	"immortal-chat/internal/adapters/repository"
	"immortal-chat/internal/config"
	"immortal-chat/internal/core/services"
)

func main() {
	fmt.Println("=== Immortal Chat OS - System Initialization (Merged Phase 2+3) ===")

	// 1. Load Configuration
	fmt.Println("[1/5] Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	fmt.Printf("✓ Config loaded (DB: %s@%s:%d)\n", cfg.DB.User, cfg.DB.Host, cfg.DB.Port)

	// 2. Connect to MariaDB (Retry Logic)
	fmt.Println("[2/5] Connecting to MariaDB...")
	db := connectMariaDB(cfg.DB, 5, 2*time.Second)
	defer db.Close()
	fmt.Println("✓ MariaDB connection established")

	// 3. Connect to Redis (Retry Logic)
	fmt.Println("[3/5] Connecting to Redis...")
	rdb := connectRedis(cfg.Redis, 5, 2*time.Second)
	defer rdb.Close()
	fmt.Println("✓ Redis connection established")

	// ==================================================================
	// INIT ARCHITECTURE LAYERS
	// ==================================================================
	fmt.Println("[4/5] Initializing Layers...")

	// A. Repositories
	mariadbRepo := repository.NewMariaDBRepository(db)
	redisRepo := repository.NewRedisRepository(rdb)

	// B. Services (Gateway is instantiated inside handlers as needed)
	dispatcher := services.NewDispatcher(
		mariadbRepo,
		mariadbRepo,
		mariadbRepo,
		redisRepo,
	)

	// D. Handlers
	webhookHandler := handler.NewWebhookHandler(
		dispatcher,
		cfg.Facebook.AppSecret,
		cfg.Facebook.VerifyToken,
	)

	// Dashboard Handler (Phase 3 Upgrade)
	// Lưu ý: DashboardHandler cần hỗ trợ cả method cũ (Metrics) và mới (Chat)
	dashboardHandler := handler.NewDashboardHandler(db, rdb)

	// ==================================================================
	// ROUTING SETUP (FIX LỖI STATIC FILES & 404)
	// ==================================================================
	fmt.Println("[5/5] Configuring Routes...")

	mux := http.NewServeMux()

	// 1. STATIC FILES (FIX LỖI QUAN TRỌNG)
	// Map request bắt đầu bằng /static/ vào thư mục ./web/static/
	// Điều này giúp tải file JS/CSS chính xác thay vì trả về HTML
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "web", "static")
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. PHASE 2 API (GIỮ NGUYÊN TÍNH NĂNG CŨ)
	mux.HandleFunc("/api/status", dashboardHandler.GetStatus)
	mux.HandleFunc("/api/system/metrics", dashboardHandler.GetSystemMetrics)
	mux.HandleFunc("/api/platforms", dashboardHandler.GetPlatforms)     // <-- Đã khôi phục
	mux.HandleFunc("/api/sync/status", dashboardHandler.GetSyncStatus) // <-- Đã khôi phục

	// 3. PHASE 3 API (TÍNH NĂNG CHAT MỚI)
	mux.HandleFunc("/api/conversations", dashboardHandler.GetConversations)
	
	// Route con cho messages (VD: /api/conversations/123/messages)
	mux.HandleFunc("/api/conversations/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/messages") {
			dashboardHandler.GetConversationMessages(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	
	mux.HandleFunc("/api/messages/reply", dashboardHandler.SendReply)

	// 4. FACEBOOK WEBHOOK
	mux.HandleFunc("/webhook/facebook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			webhookHandler.HandleFacebookVerify(w, r)
		} else if r.Method == http.MethodPost {
			webhookHandler.HandleFacebookEvent(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// 5. ROOT HANDLER (SPA Fallback)
	// Tất cả request không khớp API hay Static sẽ trả về index.html (để React/JS xử lý)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Nếu cố tình gọi file không tồn tại (vd: /js/missing.js) thì trả về 404
		// chứ không trả về index.html (tránh lỗi cú pháp <)
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	// ==================================================================
	// START SERVER
	// ==================================================================
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	fmt.Printf("\n✅ [READY] Server listening on %s\n", addr)
	fmt.Println("👉 Dashboard: http://localhost:8080/")
	fmt.Println("👉 Static Dir mapped to:", staticDir)

	// Start Watchdog Service (Phase 2 Resilience)
	services.RunWatchdog(db)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("❌ HTTP server failed: %v", err)
	}
}

// --- Helper Functions (Logic Retry không đổi) ---

func connectMariaDB(cfg config.DBConfig, maxRetries int, retryDelay time.Duration) *sql.DB {
	dsn := cfg.GetDSN()
	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("  Attempt %d/%d: Driver config error: %v", i, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}
		if err = db.Ping(); err == nil {
			return db // Success
		}
		log.Printf("  Attempt %d/%d: Ping failed: %v", i, maxRetries, err)
		db.Close()
		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	log.Fatalf("❌ Cannot connect to MariaDB after %d attempts", maxRetries)
	return nil
}

func connectRedis(cfg config.RedisConfig, maxRetries int, retryDelay time.Duration) *redis.Client {
	rdb := redis.NewClient(&redis.Options{Addr: cfg.Addr})
	ctx := context.Background()

	for i := 1; i <= maxRetries; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			return rdb // Success
		}
		log.Printf("  Attempt %d/%d: Redis ping failed", i, maxRetries)
		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	log.Fatalf("❌ Cannot connect to Redis after %d attempts", maxRetries)
	return nil
}