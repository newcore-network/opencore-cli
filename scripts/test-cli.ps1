$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Invoke-Checked {
    param([scriptblock]$Command, [string]$FailureMessage)
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw $FailureMessage
    }
}

Write-Host "🧪 Testing OpenCore CLI" -ForegroundColor Cyan
Write-Host ""

# Build CLI
Write-Host "📦 Building CLI..." -ForegroundColor Yellow
Invoke-Checked { go build -o opencore.exe . } "Build failed"
Write-Host "✅ Build successful" -ForegroundColor Green
Write-Host ""

# Test version
Write-Host "📋 Testing version command..." -ForegroundColor Yellow
Invoke-Checked { .\opencore.exe --version } "Version command failed"
Write-Host ""

# Test help
Write-Host "📋 Testing help command..." -ForegroundColor Yellow
Invoke-Checked { .\opencore.exe --help } "Help command failed"
Write-Host ""

# Test doctor (will fail if not in OpenCore project)
Write-Host "📋 Testing doctor command..." -ForegroundColor Yellow
& .\opencore.exe doctor
Write-Host ""

# Create test project
$testDir = "test-project-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
Write-Host "📋 Testing init command (creating $testDir)..." -ForegroundColor Yellow
Invoke-Checked { .\opencore.exe init $testDir } "Init failed"
Write-Host "✅ Init successful" -ForegroundColor Green
Write-Host ""

# Navigate to test project
Push-Location $testDir
try {

# Test create feature
Write-Host "📋 Testing create feature command..." -ForegroundColor Yellow
Invoke-Checked { ..\opencore.exe create feature banking } "Create feature failed"
Write-Host "✅ Create feature successful" -ForegroundColor Green
Write-Host ""

# Test create resource
Write-Host "📋 Testing create resource command..." -ForegroundColor Yellow
Invoke-Checked { ..\opencore.exe create resource chat --with-client } "Create resource failed"
Write-Host "✅ Create resource successful" -ForegroundColor Green
Write-Host ""

# Return to original directory
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "✅ All tests passed!" -ForegroundColor Green
Write-Host ""
Write-Host "Test project created at: $testDir" -ForegroundColor Cyan
Write-Host "To clean up: Remove-Item -Recurse -Force $testDir" -ForegroundColor Gray
