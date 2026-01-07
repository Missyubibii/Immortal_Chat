#!/bin/bash

#echo "🚀 [1/4] Dừng tạm cất các thay đổi cục bộ (docker-compose...)..."
#git stash push docker-compose.yml

echo "🚀 [1/3] Đang dọn dẹp và chuẩn bị..."
git add .

echo "📥 [2/4] Đang kéo code mới từ GitHub..."
git pull origin main

#echo "📤 [3/4] Đang lấy lại cấu hình docker-compose của bạn..."
#git stash pop

echo "🏗️ [3/3] Đang Build và khởi động lại Docker..."
# Kiểm tra lệnh docker compose (v2) hoặc docker-compose (v1)
if docker compose version >/dev/null 2>&1; then
    docker compose up -d --build --remove-orphans
else
    docker-compose up -d --build --remove-orphans
fi

echo "✅ Cập nhật hoàn tất!"
