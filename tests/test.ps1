# Test script for Terminal Intelligence
# Usage:
#   .\test.ps1           - Run all tests (fast mode)
#   .\test.ps1 -Full     - Run all tests including slow property-based tests
#   .\test.ps1 -Package <name> - Run tests for specific package

param(
    [switch]$Full,
    [string]$Package = "./..."
)

Write-Host "Running Terminal Intelligence Tests" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

if ($Full) {
    Write-Host "Running FULL test suite (including slow property-based tests)..." -ForegroundColor Yellow
    go test $Package -v -timeout 5m
} else {
    Write-Host "Running FAST test suite (skipping slow property-based tests)..." -ForegroundColor Green
    Write-Host "Use -Full flag to run complete test suite" -ForegroundColor Gray
    Write-Host ""
    go test $Package -short -v
}

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "✓ All tests passed!" -ForegroundColor Green
} else {
    Write-Host ""
    Write-Host "✗ Some tests failed" -ForegroundColor Red
    exit 1
}
