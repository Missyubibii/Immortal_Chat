# ============================================================================
# IMMORTAL CHAT OS - Directory Structure Initialization Script
# ============================================================================
# Purpose: Create Hexagonal Architecture directory structure
# Platform: Windows PowerShell
# Usage: .\init_structure.ps1
# ============================================================================

Write-Host "[*] Initializing Immortal Chat OS Directory Structure..." -ForegroundColor Cyan
Write-Host ""

# Define all directories following Hexagonal Architecture
$directories = @(
    "cmd\server",
    "internal\core\domain",
    "internal\core\ports",
    "internal\core\services",
    "internal\adapters\handler",
    "internal\adapters\repository",
    "internal\adapters\websocket",
    "internal\config",
    "migrations"
)

# Create each directory
foreach ($dir in $directories) {
    if (Test-Path $dir) {
        Write-Host "[OK] Directory already exists: $dir" -ForegroundColor Yellow
    } else {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        Write-Host "[+] Created directory: $dir" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "[SUCCESS] Directory structure initialized successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Structure created:" -ForegroundColor Cyan
Write-Host "   cmd/server/           - Application entry point" -ForegroundColor Gray
Write-Host "   internal/core/        - Business logic (domain, ports, services)" -ForegroundColor Gray
Write-Host "   internal/adapters/    - External interfaces (handler, repository, websocket)" -ForegroundColor Gray
Write-Host "   internal/config/      - Configuration management" -ForegroundColor Gray
Write-Host "   migrations/           - Database schema files" -ForegroundColor Gray
Write-Host ""
