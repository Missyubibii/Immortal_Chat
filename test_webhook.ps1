# ============================================================================
# IMMORTAL CHAT OS - Facebook Webhook Test Script
# ============================================================================
# Purpose: Send a test webhook with valid HMAC-SHA256 signature
# Platform: Windows PowerShell
# ============================================================================

Write-Host "=== Facebook Webhook Test Script ===" -ForegroundColor Cyan
Write-Host ""

# 1. Read APP_SECRET from .env file
$envFile = ".env"
if (-not (Test-Path $envFile)) {
    Write-Host "ERROR: .env file not found!" -ForegroundColor Red
    Write-Host "Please create .env file from .env.example" -ForegroundColor Yellow
    exit 1
}

$appSecret = ""
Get-Content $envFile | ForEach-Object {
    if ($_ -match "^FB_APP_SECRET=(.+)$") {
        $appSecret = $matches[1].Trim()
    }
}

if ([string]::IsNullOrEmpty($appSecret)) {
    Write-Host "ERROR: FB_APP_SECRET not found in .env file!" -ForegroundColor Red
    exit 1
}

Write-Host "[OK] Found FB_APP_SECRET: $($appSecret.Substring(0, [Math]::Min(10, $appSecret.Length)))..." -ForegroundColor Green

# 2. Configuration
$url = "http://localhost:8080/webhook/facebook"

# 3. Create test webhook payload (valid Facebook format)
$bodyJson = @"
{
  "object": "page",
  "entry": [{
    "id": "123456789",
    "time": 1234567890,
    "messaging": [{
      "sender": {"id": "USER_PSID_TEST_123"},
      "recipient": {"id": "PAGE_ID_456"},
      "timestamp": 1234567890,
      "message": {
        "mid": "mid.test_$(Get-Date -Format 'yyyyMMddHHmmss')",
        "text": "Xin chào! Đây là tin nhắn test từ PowerShell"
      }
    }]
  }]
}
"@

Write-Host "[INFO] Payload prepared ($(($bodyJson -replace '\s').Length) bytes)" -ForegroundColor Cyan

# 4. Calculate HMAC-SHA256 signature (exact Facebook format)
Write-Host "[INFO] Calculating HMAC-SHA256 signature..." -ForegroundColor Cyan

$hmacsha = New-Object System.Security.Cryptography.HMACSHA256
$hmacsha.Key = [Text.Encoding]::UTF8.GetBytes($appSecret)
$payloadBytes = [Text.Encoding]::UTF8.GetBytes($bodyJson)
$signature = $hmacsha.ComputeHash($payloadBytes)
$signatureHex = -join ($signature | ForEach-Object { "{0:x2}" -f $_ })
$headerSignature = "sha256=" + $signatureHex

Write-Host "[OK] Signature: sha256=$($signatureHex.Substring(0, 16))..." -ForegroundColor Green

# 5. Send POST request to webhook
Write-Host ""
Write-Host "Sending webhook to: $url" -ForegroundColor Yellow
Write-Host ""

try {
    $response = Invoke-RestMethod -Uri $url `
        -Method POST `
        -ContentType "application/json" `
        -Body $bodyJson `
        -Headers @{
            "X-Hub-Signature-256" = $headerSignature
        }
    
    Write-Host "SUCCESS! Response: $response" -ForegroundColor Green
    Write-Host ""
    Write-Host "Check logs to verify message was processed:" -ForegroundColor Cyan
    Write-Host "  docker logs chat_os_app --tail 20" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Check database:" -ForegroundColor Cyan
    Write-Host '  docker exec chat_os_db mariadb -uroot -proot_password immortal_chat -e "SELECT * FROM messages ORDER BY created_at DESC LIMIT 1;"' -ForegroundColor Gray
    
} catch {
    Write-Host "FAILED! Error: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    Write-Host "Response: $($_.Exception.Response)" -ForegroundColor Yellow
    
    if ($_.Exception.Message -match "Invalid signature") {
        Write-Host ""
        Write-Host "Troubleshooting:" -ForegroundColor Yellow
        Write-Host "1. Check that FB_APP_SECRET in .env matches what you're using" -ForegroundColor Gray
        Write-Host "2. Restart Docker containers: docker-compose restart app" -ForegroundColor Gray
        Write-Host "3. Check app logs: docker logs chat_os_app" -ForegroundColor Gray
    }
    
    exit 1
}
