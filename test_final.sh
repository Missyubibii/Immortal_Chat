# =================================================================
# CẤU HÌNH (Sửa lại cho khớp với file .env của bạn)
# =================================================================
APP_SECRET="29c661b063722d261491a7adbae89043"  # <== THAY APP SECRET CỦA BẠN VÀO ĐÂY
URL="http://localhost/webhook/facebook"
PAGE_ID="770225079500025"                       # <== PAGE ID TRONG DB

# =================================================================
# TẠO PAYLOAD JSON
# =================================================================
TIMESTAMP=$(date +%s%3N)
PAYLOAD=$(cat <<JSON
{
  "object": "page",
  "entry": [
    {
      "id": "$PAGE_ID",
      "time": $TIMESTAMP,
      "messaging": [
        {
          "sender": { "id": "KHACH_HANG_LINUX_01" },
          "recipient": { "id": "$PAGE_ID" },
          "timestamp": $TIMESTAMP,
          "message": {
            "mid": "mid.linux.$TIMESTAMP",
            "text": "Hello from Linux Bash Script!"
          }
        }
      ]
    }
  ]
}
JSON
)

# =================================================================
# TÍNH CHỮ KÝ HMAC-SHA256 (QUAN TRỌNG)
# =================================================================
# Dùng OpenSSL để tạo chữ ký giống hệt cách Facebook làm
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$APP_SECRET" | sed 's/^.* //')

echo "------------------------------------------------"
echo "🚀 Đang gửi tin nhắn test từ Linux..."
echo "👉 URL: $URL"
echo "👉 Signature: sha256=$SIGNATURE"
echo "------------------------------------------------"

# =================================================================
# GỬI REQUEST BẰNG CURL
# =================================================================
curl -X POST "$URL" \
     -H "Content-Type: application/json" \
     -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
     -d "$PAYLOAD"

echo ""
echo "------------------------------------------------"
EOF