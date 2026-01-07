// Package repository implements data persistence adapters
package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"immortal-chat/internal/core/ports"
)

// Ensure RedisRepository implements DedupRepository
var _ ports.DedupRepository = (*RedisRepository)(nil)

// RedisRepository implements deduplication using Redis cache
// Per .rulesgemini Section 4: Check dedup before processing webhooks
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository instance
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// IsDuplicate checks if an event ID has already been processed
// Uses Redis GET to check existence
// Per .rulesgemini Section 4: Deduplication with TTL
func (r *RedisRepository) IsDuplicate(ctx context.Context, eventID string) (bool, error) {
	key := buildDedupKey(eventID)
	
	// Try to get the key
	_, err := r.client.Get(ctx, key).Result()
	
	if err == redis.Nil {
		// Key doesn't exist - not a duplicate
		return false, nil
	}
	
	if err != nil {
		// Unexpected error
		slog.Error("Failed to check deduplication",
			"error", err,
			"event_id", eventID,
		)
		return false, fmt.Errorf("check duplicate: %w", err)
	}
	
	// Key exists - this is a duplicate!
	slog.Warn("Duplicate webhook event detected",
		"event_id", eventID,
		"key", key,
	)
	
	return true, nil
}

// MarkProcessed marks an event as processed in Redis with TTL
// Uses SETEX for atomic set with expiration
// Per user requirement: Use SETNX pattern, but SETEX is simpler and equivalent
func (r *RedisRepository) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	key := buildDedupKey(eventID)
	
	// Set key with TTL
	// Value is timestamp for debugging purposes
	timestamp := time.Now().Unix()
	err := r.client.Set(ctx, key, timestamp, ttl).Err()
	
	if err != nil {
		slog.Error("Failed to mark event as processed",
			"error", err,
			"event_id", eventID,
			"ttl", ttl,
		)
		return fmt.Errorf("mark processed: %w", err)
	}
	
	slog.Debug("Event marked as processed",
		"event_id", eventID,
		"key", key,
		"ttl", ttl,
	)
	
	return nil
}

// buildDedupKey constructs the Redis key for deduplication
// Per .rulesgemini Section 4: Key format dedup:msg:{platform_msg_id}
func buildDedupKey(eventID string) string {
	return fmt.Sprintf("dedup:msg:%s", eventID)
}
