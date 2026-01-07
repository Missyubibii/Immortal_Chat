# ============================================================================
# Simple Facebook Webhook Test (Corrected)
# ============================================================================

# Read actual secret from .env
$appSecret = ""
Get-Content ".env" | ForEach-Object {
    if ($_ -match "^FB_APP_SECRET=(.+)$") {
        $appSecret = $matches[1].Trim()
    }
}

Write-Host "Using secret: $appSecret" -ForegroundColor Green

# Simple single-line JSON (no formatting issues)
$bodyJson = '{"object":"page","entry":[{"id":"123456","time":1234567890,"messaging":[{"sender":{"id":"USER_TEST_123"},"recipient":{"id":"PAGE_456"},"timestamp":1234567890,"message":{"mid":"mid.test_unique_123","text":"Hello from PowerShell test"}}]}]}'

Write-Host "JSON payload: $($bodyJson.Substring(0, 80))..." -ForegroundColor Cyan

# Generate HMAC-SHA256 signature
$hmac = New-Object System.Security.Cryptography.HMACSHA256
$hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($appSecret)
$hash = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($bodyJson))
$signature = "sha256=" + (-join ($hash | ForEach-Object { $_.ToString("x2") }))

Write-Host "Signature: $signature" -ForegroundColor Yellow

# Send request
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/webhook/facebook" `
        -Method POST `
        -ContentType "application/json; charset=utf-8" `
        -Body $bodyJson `
        -Headers @{"X-Hub-Signature-256" = $signature} `
        -UseBasicParsing
    
    Write-Host ""
    Write-Host "SUCCESS! Response: $($response.Content)" -ForegroundColor Green
    Write-Host "Status: $($response.StatusCode)" -ForegroundColor Green
    
} catch {
    Write-Host ""
    Write-Host "FAILED: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Status: $($_.Exception.Response.StatusCode)" -ForegroundColor Red
}
