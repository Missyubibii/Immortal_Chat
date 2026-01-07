# ============================================================================
# TEST WEBHOOK - Immortal Chat OS
# ============================================================================

Write-Host "=== Testing Facebook Webhook ===" -ForegroundColor Cyan

# 1. Lấy secret từ .env
$appSecret = ""
Get-Content ".env" | ForEach-Object {
    if ($_ -match "^FB_APP_SECRET=(.+)$") {
        $appSecret = $matches[1].Trim()
    }
}

if ([string]::IsNullOrEmpty($appSecret)) {
    Write-Host "ERROR: Không tìm thấy FB_APP_SECRET trong .env!" -ForegroundColor Red
    exit 1
}

Write-Host "Secret: $($appSecret.Substring(0,20))..." -ForegroundColor Green

# 2. Tạo payload CHUẨN Facebook (có mid, timestamp)
$timestamp = [int][double]::Parse((Get-Date -UFormat %s))
$mid = "mid.test_$(Get-Date -Format 'HHmmss')"

$bodyJson = @"
{"object":"page","entry":[{"id":"123456","time":$timestamp,"messaging":[{"sender":{"id":"USER_TEST_VIETNAM"},"recipient":{"id":"PAGE_IMMORTAL"},"timestamp":$timestamp,"message":{"mid":"$mid","text":"Xin chào! Test từ PowerShell"}}]}]}
"@

Write-Host "Payload ready" -ForegroundColor Cyan

# 3. Tính HMAC-SHA256 signature
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($appSecret)
$payloadBytes = [System.Text.Encoding]::UTF8.GetBytes($bodyJson)
$hash = $hmac.ComputeHash($payloadBytes)
$signature = "sha256=" + (-join ($hash | ForEach-Object { $_.ToString("x2") }))

Write-Host "Signature: $($signature.Substring(0,40))..." -ForegroundColor Yellow

# 4. Gửi request
Write-Host ""
Write-Host "Sending to http://localhost:8080/webhook/facebook ..." -ForegroundColor White

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/webhook/facebook" `
        -Method POST `
        -ContentType "application/json; charset=utf-8" `
        -Body $bodyJson `
        -Headers @{"X-Hub-Signature-256" = $signature} `
        -UseBasicParsing
    
    Write-Host ""
    Write-Host "✅ SUCCESS!" -ForegroundColor Green
    Write-Host "Response: $($response.Content)" -ForegroundColor White
    Write-Host ""
    
    Write-Host "Kiểm tra database:" -ForegroundColor Cyan
    Write-Host '  docker exec chat_os_db mariadb -u admin -pQvc@1011 immortal_chat -e "SELECT * FROM messages ORDER BY id DESC LIMIT 1\G"' -ForegroundColor Gray
    
    Start-Sleep -Seconds 2
    Write-Host ""
    Write-Host "Fetching from database..." -ForegroundColor Cyan
    docker exec chat_os_db mariadb -u admin -pQvc@1011 immortal_chat -e "SELECT id, sender_id, content, created_at FROM messages ORDER BY id DESC LIMIT 1;"
    
} catch {
    Write-Host ""
    Write-Host "❌ FAILED!" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Yellow
    
    if ($_.Exception.Message -match "Invalid signature") {
        Write-Host ""
        Write-Host "Debug info:" -ForegroundColor Yellow
        Write-Host "  Secret used: $appSecret" -ForegroundColor Gray
        Write-Host "  Payload length: $($bodyJson.Length) bytes" -ForegroundColor Gray
        Write-Host ""
        Write-Host "Check container secret:" -ForegroundColor Cyan
        docker exec chat_os_app printenv FB_APP_SECRET
    }
}
