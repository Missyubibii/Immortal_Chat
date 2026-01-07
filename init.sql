CREATE TABLE IF NOT EXISTS tenants (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    plan VARCHAR(50) DEFAULT 'basic',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pages (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    platform VARCHAR(20) DEFAULT 'facebook',
    page_id VARCHAR(50) NOT NULL UNIQUE,     -- ID Page lấy từ Facebook
    page_name VARCHAR(255),
    access_token TEXT,                       -- Token dài hạn để gửi tin lại
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- 3. Tạo bảng Conversations (Cập nhật thêm để đảm bảo logic chạy đúng)
CREATE TABLE IF NOT EXISTS conversations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id INT NOT NULL,
    platform_id VARCHAR(100) NOT NULL, -- ID người chat (PSID)
    page_id VARCHAR(100),              -- ID Page nhận tin
    customer_name VARCHAR(255),
    status VARCHAR(20) DEFAULT 'unread',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_conversation (tenant_id, platform_id, page_id)
);
