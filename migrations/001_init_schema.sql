-- ============================================================================
-- IMMORTAL CHAT OS - Initial Database Schema
-- ============================================================================
-- Migration: 001_init_schema
-- Purpose: Create core tables for federated chat system
-- CRITICAL: Configure distributed ID settings to prevent Master-Master conflicts
-- ============================================================================

-- Configure Distributed ID Strategy (MANDATORY per .rulesgemini Section 1)
-- This prevents AUTO_INCREMENT conflicts when syncing data across nodes
SET GLOBAL auto_increment_increment = 10;  -- Allow up to 10 nodes in the mesh
SET GLOBAL auto_increment_offset = 1;      -- This is Node #1

-- ============================================================================
-- Table: webhook_logs
-- Purpose: Store incoming webhook payloads for audit and replay
-- ============================================================================
CREATE TABLE IF NOT EXISTS webhook_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    platform VARCHAR(50) NOT NULL COMMENT 'Source platform: facebook, zalo, telegram',
    payload TEXT NOT NULL COMMENT 'Raw JSON payload from webhook',
    status VARCHAR(20) DEFAULT 'pending' COMMENT 'processing status: pending, processed, failed',
    is_synced BOOLEAN DEFAULT 0 COMMENT 'Sync status to Home Server (0=local only, 1=synced)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_sync (is_synced),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Table: conversations
-- Purpose: Chat conversation metadata (group/1-1)
-- ============================================================================
CREATE TABLE IF NOT EXISTS conversations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    platform VARCHAR(50) NOT NULL COMMENT 'Platform: facebook, zalo, telegram',
    external_id VARCHAR(255) NOT NULL COMMENT 'Platform-specific conversation ID',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_platform_conversation (platform, external_id),
    INDEX idx_platform (platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Table: messages
-- Purpose: Core message storage (Local-First architecture)
-- ============================================================================
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id BIGINT NOT NULL COMMENT 'Reference to conversations.id',
    sender_id VARCHAR(255) NOT NULL COMMENT 'Platform-specific sender ID',
    content TEXT NOT NULL COMMENT 'Message content',
    is_synced BOOLEAN DEFAULT 0 COMMENT 'Sync status to Home Server (0=local only, 1=synced)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_conversation (conversation_id),
    INDEX idx_sync (is_synced),
    INDEX idx_created (created_at),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================================================
-- Verify Schema Creation
-- ============================================================================
-- You can verify tables with: SHOW TABLES;
-- Check structure with: DESCRIBE webhook_logs; DESCRIBE messages; DESCRIBE conversations;
