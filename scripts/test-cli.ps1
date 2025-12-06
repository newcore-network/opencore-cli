# Test script for OpenCore CLI
# Run this to test all commands locally

Write-Host "🧪 Testing OpenCore CLI" -ForegroundColor Cyan
Write-Host ""

# Build CLI
Write-Host "📦 Building CLI..." -ForegroundColor Yellow
go build -o opencore.exe ./cmd/opencore
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Build successful" -ForegroundColor Green
Write-Host ""

# Test version
Write-Host "📋 Testing version command..." -ForegroundColor Yellow
.\opencore.exe --version
Write-Host ""

# Test help
Write-Host "📋 Testing help command..." -ForegroundColor Yellow
.\opencore.exe --help
Write-Host ""

# Test doctor (will fail if not in OpenCore project)
Write-Host "📋 Testing doctor command..." -ForegroundColor Yellow
.\opencore.exe doctor
Write-Host ""

# Create test project
$testDir = "test-project-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
Write-Host "📋 Testing init command (creating $testDir)..." -ForegroundColor Yellow
.\opencore.exe init $testDir
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Init failed" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Init successful" -ForegroundColor Green
Write-Host ""

# Navigate to test project
Push-Location $testDir

# Test create feature
Write-Host "📋 Testing create feature command..." -ForegroundColor Yellow
..\opencore.exe create feature banking
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Create feature failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "✅ Create feature successful" -ForegroundColor Green
Write-Host ""

# Test create resource
Write-Host "📋 Testing create resource command..." -ForegroundColor Yellow
..\opencore.exe create resource chat --with-client
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Create resource failed" -ForegroundColor Red
    Pop-Location
    exit 1
}
Write-Host "✅ Create resource successful" -ForegroundColor Green
Write-Host ""

# Return to original directory
Pop-Location

Write-Host ""
Write-Host "✅ All tests passed!" -ForegroundColor Green
Write-Host ""
Write-Host "Test project created at: $testDir" -ForegroundColor Cyan
Write-Host "To clean up: Remove-Item -Recurse -Force $testDir" -ForegroundColor Gray

