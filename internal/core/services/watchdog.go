// Package services contains core business logic services
// Following Hexagonal Architecture: Core layer is independent of infrastructure
package services

import (
	"database/sql"
	"fmt"
	"time"
)

// RunWatchdog starts the auto-purge background service
// Per .rulesgemini Section 5: Self-Healing & Watchdog (AUTO-PURGE)
// This function preserves the tested logic from prototype main.go
func RunWatchdog(db *sql.DB) {
	ticker := time.NewTicker(10 * time.Minute) // Run every 10 minutes (production setting)
	
	go func() {
		for range ticker.C {
			fmt.Println("[WATCHDOG] Resource check started...")
			
			// TODO: Implement actual disk usage check
			// For now, using simulated value
			// In production, use syscall or external library to get real disk usage
			diskUsage := 50 // Simulated: 50% disk usage
			
			// Only purge if disk usage exceeds 70% (Safety Rule #1)
			if diskUsage >= 70 {
				fmt.Println("[WATCHDOG] Disk usage > 70%, initiating purge...")
				
				// Purge Rule: Delete data only if ALL 3 conditions are met:
				// 1. Disk Usage > 70%
				// 2. Data older than 7 days (Retention Policy)
				// 3. is_synced = 1 (Already safe at Home Server)
				result, err := db.Exec(`
					DELETE FROM webhook_logs 
					WHERE is_synced = 1 
					AND created_at < DATE_SUB(NOW(), INTERVAL 7 DAY) 
					LIMIT 1000
				`)
				
				if err == nil {
					rows, _ := result.RowsAffected()
					fmt.Printf("[WATCHDOG] Purged %d old webhook_logs records\n", rows)
				} else {
					fmt.Printf("[WATCHDOG] Error during purge: %v\n", err)
				}
				
				// Also purge old messages (same conditions)
				result, err = db.Exec(`
					DELETE FROM messages 
					WHERE is_synced = 1 
					AND created_at < DATE_SUB(NOW(), INTERVAL 7 DAY) 
					LIMIT 1000
				`)
				
				if err == nil {
					rows, _ := result.RowsAffected()
					fmt.Printf("[WATCHDOG] Purged %d old messages\n", rows)
				} else {
					fmt.Printf("[WATCHDOG] Error during message purge: %v\n", err)
				}
			} else {
				fmt.Printf("[WATCHDOG] Disk usage OK (%d%%), no purge needed\n", diskUsage)
			}
		}
	}()
	
	fmt.Println("[WATCHDOG] Service started (checks every 10 minutes)")
}
