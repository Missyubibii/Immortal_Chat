# 🚀 Hướng Dẫn Test Webhook Facebook Thực Tế

## Mục tiêu

Nhận tin nhắn từ Facebook Page thực tế về server của bạn

---

## Bước 1: Expose Server Ra Internet

Server hiện đang chạy trên `localhost:8080`, Facebook không thể kết nối được. Bạn có 3 lựa chọn:

### Lựa Chọn 1: Dùng Cloudflare Tunnel (Khuyên Dùng - Free & An Toàn)

```bash
# Cài đặt cloudflared (nếu chưa có)
winget install --id Cloudflare.cloudflared

# Tạo tunnel nhanh (quick tunnel)
cloudflared tunnel --url http://localhost:8080
```

**Output mẫu**:

```
Your quick Tunnel has been created! Visit it at:
https://abc-def-123.trycloudflare.com
```

**⚠️ Lưu lại URL này!** Bạn sẽ dùng nó làm Webhook URL.

---

### Lựa Chọn 2: Dùng ngrok (Miễn Phí)

```bash
# Download từ https://ngrok.com/download
# Chạy:
ngrok http 8080
```

**Output**:

```
Forwarding https://abc123.ngrok.io -> http://localhost:8080
```

---

### Lựa Chọn 3: Deploy lên VPS (Production)

Nếu có VPS với IP public, cấu hình nginx reverse proxy.

---

## Bước 2: Chuẩn Bị Facebook App

### 2.1. Tạo Facebook App

1. Truy cập: https://developers.facebook.com/apps/
2. Nhấn **"Create App"**
3. Chọn **"Business"** → **"Next"**
4. Nhập tên app: `Immortal Chat Test`
5. Email liên hệ: email của bạn
6. Nhấn **"Create App"**

### 2.2. Thêm Messenger Product

1. Trong Dashboard, tìm **"Add Products"**
2. Chọn **Messenger** → **"Set Up"**

### 2.3. Lấy Thông Tin Quan Trọng

#### App Secret

1. **Settings** → **Basic**
2. Nhấn **"Show"** ở mục **"App Secret"**
3. Copy giá trị → Dán vào `.env`:

```env
FB_APP_SECRET=abc123def456...  # Thay bằng App Secret thực
```

#### Verify Token (Tự Đặt)

Đây là mật khẩu bạn tự chọn để verify webhook:

```env
FB_VERIFY_TOKEN=immortal_chat_secure_token_2026
```

**⚠️ Sau khi sửa .env, restart app**:

```bash
docker-compose restart app
```

---

## Bước 3: Kết Nối Page Với App

### 3.1. Tạo/Chọn Facebook Page

1. Tạo Page test tại: https://www.facebook.com/pages/create/
2. Hoặc dùng Page có sẵn

### 3.2. Add Page To App

1. Vào **Messenger** → **Settings**
2. Phần **"Access Tokens"**
3. Nhấn **"Add or Remove Pages"**
4. Chọn Page của bạn → **"Next"** → **"Done"**

---

## Bước 4: Thiết Lập Webhook

### 4.1. Cấu Hình Webhook URL

1. Trong **Messenger** → **Settings**
2. Phần **"Webhooks"** → **"Add Callback URL"**

**Điền thông tin**:

```
Callback URL: https://your-tunnel-url.trycloudflare.com/webhook/facebook
Verify Token: immortal_chat_secure_token_2026
```

_(Thay `your-tunnel-url` bằng URL thực từ Bước 1)_

3. Nhấn **"Verify and Save"**

**✅ Nếu thành công**: Bạn sẽ thấy dấu tích xanh!

**❌ Nếu lỗi**:

- Check app logs: `docker logs chat_os_app --tail 20`
- Verify Token có khớp trong `.env` không
- Tunnel còn chạy không (cloudflared/ngrok)

### 4.2. Subscribe To Page Events

1. Sau khi verify thành công
2. Phần **"Webhooks"** → chọn Page vừa thêm
3. Nhấn **"Subscribe"**
4. Tick vào: **`messages`** và **`messaging_postbacks`**
5. Nhấn **"Subscribe"**

---

## Bước 5: TEST THỰC TẾ! 🎉

### 5.1. Gửi Tin Nhắn Test

1. Mở Facebook Page của bạn
2. Nhắn tin cho chính Page đó (dùng tài khoản cá nhân)
3. Gửi: **"Hello from real Facebook!"**

### 5.2. Kiểm Tra Logs

```bash
# Xem logs real-time
docker logs -f chat_os_app
```

**Logs mong đợi**:

```
INFO Webhook received and queued for processing
INFO Message processed successfully message_id=mid.xxx sender_id=123456789
```

### 5.3. Kiểm Tra Database

```bash
docker exec chat_os_db mariadb -u admin -pQvc@1011 immortal_chat -e "SELECT id, sender_id, content, created_at FROM messages ORDER BY id DESC LIMIT 5;"
```

**Kết quả mong đợi**:

```
id  sender_id     content                        created_at
1   1234567890    Hello from real Facebook!      2026-01-07 07:50:00
```

---

## Troubleshooting

### Lỗi: "Webhook verification failed"

**Nguyên nhân**: Verify Token không khớp

**Giải pháp**:

```bash
# 1. Check token trong .env
cat .env | grep FB_VERIFY_TOKEN

# 2. Check token trong container
docker exec chat_os_app printenv FB_VERIFY_TOKEN

# 3. Nếu khác nhau, restart:
docker-compose restart app
```

### Lỗi: "Could not connect to URL"

**Nguyên nhân**: Tunnel đã tắt hoặc server không chạy

**Giải pháp**:

```bash
# 1. Check server đang chạy
docker ps | grep chat_os_app

# 2. Check tunnel còn sống
# (xem terminal chạy cloudflared/ngrok)

# 3. Test local trước
curl http://localhost:8080/
# Phải trả về: {"code":200,"message":"Immortal Chat OS is running"}
```

### Lỗi: "Invalid signature"

**Nguyên nhân**: App Secret sai

**Giải pháp**:

```bash
# 1. Copy lại App Secret từ Facebook Dashboard
# 2. Paste vào .env
# 3. Restart app
docker-compose restart app
```

### Tin nhắn không lưu vào database

**Debug**:

```bash
# 1. Check logs chi tiết
docker logs chat_os_app --tail 50

# 2. Check tenant tồn tại
docker exec chat_os_db mariadb -u admin -pQvc@1011 immortal_chat -e "SELECT * FROM tenants;"

# 3. Check Redis dedup
docker exec chat_os_redis redis-cli KEYS "dedup:msg:*"
```

---

## Tools Hữu Ích

### Test Webhook Verification (Manual)

```bash
# Test GET request (webhook verification)
curl "https://your-tunnel-url.com/webhook/facebook?hub.mode=subscribe&hub.verify_token=immortal_chat_secure_token_2026&hub.challenge=TEST123"

# Phải trả về: TEST123
```

### Monitor Real-Time Logs

```bash
# Terminal 1: App logs
docker logs -f chat_os_app

# Terminal 2: Database
watch -n 2 'docker exec chat_os_db mariadb -u admin -pQvc@1011 immortal_chat -e "SELECT COUNT(*) FROM messages"'
```

---

## Checklist Hoàn Chỉnh

- [ ] Tunnel đang chạy (cloudflared/ngrok)
- [ ] Server đang chạy (docker ps)
- [ ] `.env` có FB_APP_SECRET và FB_VERIFY_TOKEN đúng
- [ ] App đã restart sau khi sửa `.env`
- [ ] Facebook App đã tạo
- [ ] Page đã add vào App
- [ ] Webhook URL đã verify thành công ✅
- [ ] Đã subscribe vào `messages` events
- [ ] Gửi tin nhắn test vào Page
- [ ] Logs hiển thị "Message processed successfully"
- [ ] Database có record mới

---

## Tips

### Cloudflare Tunnel Lâu Dài

Thay vì quick tunnel (URL random), tạo tunnel cố định:

```bash
# 1. Login
cloudflared login

# 2. Tạo tunnel
cloudflared tunnel create immortal-chat

# 3. Cấu hình
cloudflared tunnel route dns immortal-chat chat.yourdomain.com

# 4. Chạy
cloudflared tunnel run immortal-chat
```

### Webhook Events Cần Subscribe

Để nhận đầy đủ:

- ✅ **messages** - Tin nhắn văn bản
- ✅ **messaging_postbacks** - Button clicks
- ⚠️ **messaging_optins** - Opt-in events
- ⚠️ **message_deliveries** - Delivery confirmations (tùy chọn)
- ⚠️ **message_reads** - Read receipts (tùy chọn)

---

**Good Luck! 🚀**

Nếu có lỗi, paste logs lên để mình debug nhé!
