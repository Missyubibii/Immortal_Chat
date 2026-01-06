// Package main - Immortal Chat OS Application Entry Point
// Following .rulesgemini Hexagonal Architecture principles
// Phase 1: Infrastructure Wiring Only (No business logic yet)
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"immortal-chat/internal/config"
	"immortal-chat/internal/core/services"
)

func main() {
	fmt.Println("=== Immortal Chat OS - Cell Infrastructure Initialization ===")

	// 1. Load Configuration from Environment
	fmt.Println("[1/4] Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}
	fmt.Printf("✓ Config loaded (DB: %s@%s:%d, Redis: %s)\n",
		cfg.DB.User, cfg.DB.Host, cfg.DB.Port, cfg.Redis.Addr)

	// 2. Connect to MariaDB with Retry Logic
	// Docker containers may not be ready immediately, so we retry
	fmt.Println("[2/4] Connecting to MariaDB...")
	db := connectMariaDB(cfg.DB, 5, 2*time.Second)
	defer db.Close()
	fmt.Println("✓ MariaDB connection established")

	// 3. Connect to Redis with Retry Logic
	fmt.Println("[3/4] Connecting to Redis...")
	rdb := connectRedis(cfg.Redis, 5, 2*time.Second)
	defer rdb.Close()
	fmt.Println("✓ Redis connection established")

	// 4. Infrastructure Ready!
	fmt.Println("\n✅ Cell Infrastructure Ready\n")

	// Start Watchdog Service (Self-Healing Auto-Purge)
	// Per .rulesgemini Section 5: Self-Healing & Watchdog
	services.RunWatchdog(db)

	// 5. Start HTTP Server (Keep process alive + health endpoint)
	// Following .rulesgemini Section 4: Webhook endpoints will be added in Phase 2
	startHTTPServer(cfg.App.Port)
}

// connectMariaDB attempts to connect to MariaDB with retry logic
// Retries are necessary because Docker containers may still be initializing
func connectMariaDB(cfg config.DBConfig, maxRetries int, retryDelay time.Duration) *sql.DB {
	dsn := cfg.GetDSN()

	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("  Attempt %d/%d: Failed to configure DB driver: %v", i, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		// Test the connection with Ping
		err = db.Ping()
		if err == nil {
			// Success!
			return db
		}

		log.Printf("  Attempt %d/%d: Cannot ping MariaDB: %v", i, maxRetries, err)
		db.Close()

		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	// All retries exhausted
	log.Fatalf("❌ Cannot connect to MariaDB after %d attempts: %v", maxRetries, err)
	return nil // unreachable
}

// connectRedis attempts to connect to Redis with retry logic
func connectRedis(cfg config.RedisConfig, maxRetries int, retryDelay time.Duration) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
	})

	ctx := context.Background()
	var err error

	for i := 1; i <= maxRetries; i++ {
		err = rdb.Ping(ctx).Err()
		if err == nil {
			// Success!
			return rdb
		}

		log.Printf("  Attempt %d/%d: Cannot ping Redis: %v", i, maxRetries, err)

		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}

	// All retries exhausted
	log.Fatalf("❌ Cannot connect to Redis after %d attempts: %v", maxRetries, err)
	return nil // unreachable
}

// startHTTPServer starts the HTTP server with basic health endpoint
// Following .rulesgemini: Standard library net/http (no heavy frameworks)
func startHTTPServer(port int) {
	// Health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"code":200,"message":"Immortal Chat OS is running","data":null}`)
	})

	// Webhook endpoints will be added in Phase 2
	// Per .rulesgemini Section 4: POST /webhook/:platform

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("[HTTP] Server listening on %s\n", addr)
	fmt.Println("[HTTP] Health check: http://localhost:8080/")
	fmt.Println("[READY] Press Ctrl+C to stop\n")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ HTTP server failed: %v", err)
	}
}
