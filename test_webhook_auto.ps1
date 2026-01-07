# ============================================================================
# Facebook Webhook Test - Auto-sync with Container
# ============================================================================

Write-Host "=== Checking Configuration ===" -ForegroundColor Cyan

# 1. Read secret from .env file
$localSecret = ""
Get-Content ".env" | ForEach-Object {
    if ($_ -match "^FB_APP_SECRET=(.+)$") {
        $localSecret = $matches[1].Trim()
    }
}

Write-Host "Local .env secret: $localSecret" -ForegroundColor Yellow

# 2. Get secret from running container
$containerSecret = docker exec chat_os_app printenv FB_APP_SECRET 2>$null
$containerSecret = $containerSecret.Trim()

Write-Host "Container secret: $containerSecret" -ForegroundColor Yellow

# 3. Check if they match
if ($localSecret -ne $containerSecret) {
    Write-Host ""
    Write-Host "WARNING: Secrets don't match!" -ForegroundColor Red
    Write-Host "The container is using a different secret than your .env file." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Solution: Restart the container to load the new .env:" -ForegroundColor Cyan
    Write-Host "  docker-compose restart app" -ForegroundColor Gray
    Write-Host ""
    
    $response = Read-Host "Use container's secret for this test? (Y/N)"
    if ($response -ne "Y") {
        Write-Host "Aborted. Please restart Docker container first." -ForegroundColor Yellow
        exit 1
    }
    
    $appSecret = $containerSecret
} else {
    Write-Host "Secrets match! Using: $containerSecret" -ForegroundColor Green
    $appSecret = $containerSecret
}

Write-Host ""
Write-Host "=== Sending Test Webhook ===" -ForegroundColor Cyan

# Payload
$bodyJson = '{"object":"page","entry":[{"id":"123456","time":1234567890,"messaging":[{"sender":{"id":"USER_TEST_123"},"recipient":{"id":"PAGE_456"},"timestamp":1234567890,"message":{" mid":"mid.test_' + (Get-Date -Format 'HHmmss') + '","text":"Xin chào từ PowerShell!"}}]}]}'

# Generate signature
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($appSecret)
$hash = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($bodyJson))
$signature = "sha256=" + (-join ($hash | ForEach-Object { $_.ToString("x2") }))

Write-Host "Sending to: http://localhost:8080/webhook/facebook"
Write-Host "Signature: $($signature.Substring(0, 40))..."

# Send request
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
    Write-Host "Check the results:" -ForegroundColor Cyan
    Write-Host '  docker exec chat_os_db mariadb -uroot -proot_password immortal_chat -e "SELECT id, sender_id, content, created_at FROM messages ORDER BY created_at DESC LIMIT 3;"' -ForegroundColor Gray
    
} catch {
    Write-Host ""
    Write-Host "❌ FAILED!" -ForegroundColor Red
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Yellow
    
    Write-Host ""
    Write-Host "Troubleshooting:" -ForegroundColor Cyan
    Write-Host "1. Check app logs: docker logs chat_os_app --tail 20" -ForegroundColor Gray
    Write-Host "2. Verify container is running: docker ps" -ForegroundColor Gray
    Write-Host "3. Check .env file matches container env" -ForegroundColor Gray
}
