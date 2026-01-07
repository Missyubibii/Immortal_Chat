// Package services contains panic mode management
package services

import (
	"log/slog"
	"sync"
	"time"
)

// PanicMode manages emergency AI shutdown
type PanicMode struct {
	mu        sync.RWMutex
	active    bool
	activatedBy string
	activatedAt time.Time
	reason      string
}

var globalPanicMode = &PanicMode{}

// IsActive returns whether panic mode is currently active
func (p *PanicMode) IsActive() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.active
}

// Enable activates panic mode (disables AI)
func (p *PanicMode) Enable(reason, activatedBy string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.active = true
	p.reason = reason
	p.activatedBy = activatedBy
	p.activatedAt = time.Now()
	
	slog.Warn("🚨 PANIC MODE ACTIVATED",
		"reason", reason,
		"activated_by", activatedBy,
	)
}

// Disable deactivates panic mode (re-enables AI)
func (p *PanicMode) Disable(deactivatedBy string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	duration := time.Since(p.activatedAt)
	
	p.active = false
	
	slog.Info("✅ PANIC MODE DEACTIVATED",
		"deactivated_by", deactivatedBy,
		"duration", duration,
	)
}

// GetStatus returns current panic mode status
func (p *PanicMode) GetStatus() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return map[string]interface{}{
		"active":       p.active,
		"reason":       p.reason,
		"activated_by": p.activatedBy,
		"activated_at": p.activatedAt,
	}
}

// GlobalPanicMode returns the global panic mode instance
func GlobalPanicMode() *PanicMode {
	return globalPanicMode
}
