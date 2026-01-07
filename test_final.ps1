# CẤU HÌNH (Thay cho đúng với file .env của bạn)
$AppSecret = "toi_la_bot_bat_tu" # <== PHẢI KHỚP VỚI .ENV TRONG DOCKER
$Url = "http://localhost:8080/webhook/facebook"

# JSON GIẢ LẬP FACEBOOK
$Payload = @{
    object = "page"
    entry = @(
        @{
            id = "770225079500025" # <== Thay ID Page trong DB của bạn vào đây
            time = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
            messaging = @(
                @{
                    sender = @{ id = "USER_KHACH_HANG_1" }
                    recipient = @{ id = "10523456789" }
                    timestamp = [DateTimeOffset]::Now.ToUnixTimeMilliseconds()
                    message = @{
                        mid = "mid.$([Guid]::NewGuid().ToString())"
                        text = "Chào Admin, đây là tin nhắn test manual!"
                    }
                }
            )
        }
    )
} | ConvertTo-Json -Depth 10

# TÍNH TOÁN CHỮ KÝ HMAC-SHA256 (QUAN TRỌNG NHẤT)
$HMACSHA256 = New-Object System.Security.Cryptography.HMACSHA256
$HMACSHA256.Key = [Text.Encoding]::UTF8.GetBytes($AppSecret)
$Bytes = [Text.Encoding]::UTF8.GetBytes($Payload)
$Hash = $HMACSHA256.ComputeHash($Bytes)
$Signature = "sha256=" + [BitConverter]::ToString($Hash).Replace("-", "").ToLower()

# GỬI REQUEST
Write-Host "Dang gui tin nhan test..." -ForegroundColor Cyan
try {
    $Response = Invoke-RestMethod -Uri $Url -Method Post -Body $Payload -ContentType "application/json" -Headers @{ "X-Hub-Signature-256" = $Signature }
    Write-Host "✅ Gửi thành công! Server trả lời: $Response" -ForegroundColor Green
} catch {
    Write-Host "❌ Gửi thất bại: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $Stream = $_.Exception.Response.GetResponseStream()
        $Reader = New-Object System.IO.StreamReader($Stream)
        Write-Host "Chi tiết lỗi: $($Reader.ReadToEnd())" -ForegroundColor Yellow
    }
}