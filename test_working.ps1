# 1. Cấu hình (Thay đúng Secret trong file .env của bạn vào đây)
$appSecret = "747db9089d1f903584198a284515e794" 
$url = "http://localhost:8080/webhook/facebook"

# 2. Tạo tin nhắn test
$bodyJson = '{"object":"page","entry":[{"id":"12345","time":123456789,"messaging":[{"sender":{"id":"USER_TEST_01"},"recipient":{"id":"PAGE_01"},"message":{"text":"HELLO DATABASE, ARE YOU THERE?"}}]}]}'

# 3. Ký tên (HMAC)
$hmacsha = New-Object System.Security.Cryptography.HMACSHA256
$hmacsha.Key = [Text.Encoding]::ASCII.GetBytes($appSecret)
$signature = "sha256=" + -join ($hmacsha.ComputeHash([Text.Encoding]::ASCII.GetBytes($bodyJson)) | ForEach-Object { "{0:x2}" -f $_ })

# 4. Bắn!
Invoke-RestMethod -Uri $url -Method POST -ContentType "application/json" -Body $bodyJson -Headers @{"X-Hub-Signature-256"=$signature}