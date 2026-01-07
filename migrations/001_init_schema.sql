-- 1. Bảng webhook_logs (Vùng đệm) [cite: 140-149]
CREATE TABLE IF NOT EXISTS webhook_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    platform VARCHAR(20) NOT NULL,
    payload_json JSON,
    status ENUM('pending', 'processed', 'failed') DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    error_log TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status)
);

-- 2. Bảng conversations (Hội thoại) [cite: 154-169]
CREATE TABLE IF NOT EXISTS conversations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    platform_id VARCHAR(50) NOT NULL,
    page_id VARCHAR(50),
    customer_name VARCHAR(100),
    last_message_content TEXT,
    last_message_at TIMESTAMP,
    tags JSON,
    assignee_id INT,
    status ENUM('unread', 'read', 'archived') DEFAULT 'unread',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_tenant_assignee (tenant_id, assignee_id),
    UNIQUE KEY uniq_platform (platform_id, page_id)
);

-- 3. Bảng messages (Tin nhắn) [cite: 174-187]
CREATE TABLE IF NOT EXISTS messages (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    sender_id VARCHAR(50),
    sender_type ENUM('user', 'bot', 'agent') NOT NULL,
    content TEXT,
    attachments JSON,
    type ENUM('text', 'image', 'file', 'sticker', 'voice'),
    is_synced BOOLEAN DEFAULT FALSE,
    external_msg_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_conv_time (conversation_id, created_at),
    INDEX idx_synced_time (is_synced, created_at)
);

-- 4. Bảng tenants (Khách hàng) [cite: 227-234]
CREATE TABLE IF NOT EXISTS tenants (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    plan ENUM('basic', 'pro', 'vip'),
    expired_at TIMESTAMP,
    config JSON,
    is_active BOOLEAN DEFAULT TRUE
);

-- 5. Bảng pages (Kết nối Facebook/Zalo) [cite: 239-248]
CREATE TABLE IF NOT EXISTS pages (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    platform ENUM('facebook', 'zalo') NOT NULL,
    page_id VARCHAR(50) NOT NULL,
    page_name VARCHAR(100),
    access_token TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE KEY uniq_page (platform, page_id)
);