-- Phase 3: Sample Data for Testing Conversation APIs
-- Run this AFTER 001_init_schema.sql

-- 1. Insert test page (required for conversations)
INSERT INTO pages (tenant_id, platform, page_id, page_name, access_token, is_active)
VALUES (
    1,
    'facebook',
    '770225079500025',
    'Test Immortal Chat Page',
    'EAAYdP0MtXtIBO5kVVBvP5ZC5EtSy4EKacAZAhYP4EVmZBKrj4gGvzSmIXNfhXe6fmPO46LYEgIx2pJKe3pHLTOXxrcgfZBySwZBmMAD6n0YvXeYKNIMAcfZBYPRdOKBQFoG2E1AxsFZCZCwRcqTu0DGxOdP6pz0eJlV652FXZA4h5gZACnGuwPxEJxZBbbVQfS4',
    TRUE
)
ON DUPLICATE KEY UPDATE 
    page_name = VALUES(page_name'),
    access_token = VALUES(access_token);

-- 2. Check if we have any conversations, if not create samples
-- (This will be auto-created by webhook, but for testing we can add manually)

-- Sample conversation 1
INSERT INTO conversations (
    tenant_id, platform_id, page_id, customer_name,
    last_message_content, last_message_at, tags, status, created_at
)
VALUES (
    1,
    'USER_TEST_VIETNAM',
    '770225079500025',
    'Khách Hàng Test',
    'Xin chào! Test từ PowerShell',
    NOW(),
    '[]',
    'unread',
    NOW() - INTERVAL 30 MINUTE
)
ON DUPLICATE KEY UPDATE
    last_message_at = VALUES(last_message_at);

-- Sample conversation 2 (older)
INSERT INTO conversations (
    tenant_id, platform_id, page_id, customer_name,
    last_message_content, last_message_at, tags, status, created_at
)
VALUES (
    1,
    'USER_KHACH_HANG_1',
    '770225079500025',
    'Nguyễn Văn A',
    'Chào Admin, đây là tin nhắn test manual!',
    NOW() - INTERVAL 1 HOUR,
    '[]',
    'read',
    NOW() - INTERVAL 2 HOUR
)
ON DUPLICATE KEY UPDATE
    last_message_at = VALUES(last_message_at);

-- Note: Messages will be created automatically via webhooks
-- Or you can insert sample messages here if needed for testing
