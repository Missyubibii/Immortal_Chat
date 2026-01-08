# Hướng dẫn khởi chạy Immortal Chat OS

## Vấn đề hiện tại

Bạn đang gặp lỗi khi mở file `index.html` trực tiếp trong trình duyệt. Các file JavaScript và CSS cần được phục vụ qua HTTP server.

## Lỗi đang gặp

```
dashboard.js:1  Uncaught SyntaxError: Unexpected token '<' (at dashboard.js:1:1)
Uncaught ReferenceError: switchView is not defined
```

**Nguyên nhân**: Bạn đang mở file HTML trực tiếp (`file:///...`) thay vì qua server (`http://localhost:8080`)

## Giải pháp

### Cách 1: Chạy Go Server (Khuyến nghị)

```powershell
# Di chuyển vào thư mục dự án
cd c:\laragon\www\ImmortalChatOS

# Chạy server
go run cmd/server/main.go
```

Sau đó truy cập: **http://localhost:8080**

### Cách 2: Chạy server đơn giản bằng PHP (Nếu Go chưa cấu hình)

```powershell
# Di chuyển vào thư mục static
cd c:\laragon\www\ImmortalChatOS\web\static

# Chạy PHP built-in server
php -S localhost:8080
```

Sau đó truy cập: **http://localhost:8080**

### Cách 3: Chạy bằng Python

```powershell
# Python 3
cd c:\laragon\www\ImmortalChatOS\web\static
python -m http.server 8080
```

Sau đó truy cập: **http://localhost:8080**

## Kiểm tra

Sau khi chạy server, mở F12 Console, các lỗi sẽ biến mất và bạn sẽ thấy dashboard hoạt động bình thường.

## Lưu ý về TailwindCSS CDN

Warning về TailwindCSS CDN chỉ là cảnh báo, không ảnh hưởng chức năng trong môi trường development. Khi deploy production, hãy cài đặt TailwindCSS theo hướng dẫn tại: https://tailwindcss.com/docs/installation
