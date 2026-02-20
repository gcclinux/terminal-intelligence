# Terminal Intelligence Build Script for Windows
param(
    [string]$Target = "build"
)

$BINARY_NAME = "ti"
$VERSION = "0.0.1.2"
$BUILD_DIR = "build"

# Get build number from git
try {
    $gitCount = git rev-list --count HEAD 2>$null
    if ($LASTEXITCODE -eq 0) {
        $BUILD_NUMBER = [int]$gitCount + 1
    } else {
        $BUILD_NUMBER = 1
    }
} catch {
    $BUILD_NUMBER = 1
}

$LDFLAGS = "-s -w -X main.version=$VERSION -X main.buildNumber=$BUILD_NUMBER"

function Build-Current {
    Write-Host "Building $BINARY_NAME for current platform..." -ForegroundColor Cyan
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME.exe" .
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build complete: $BUILD_DIR/$BINARY_NAME.exe" -ForegroundColor Green
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
}

function Build-Windows {
    Write-Host "Building $BINARY_NAME for Windows..." -ForegroundColor Cyan
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME-windows-amd64.exe" .
    Write-Host "Build complete: $BUILD_DIR/$BINARY_NAME-windows-amd64.exe" -ForegroundColor Green
}

function Build-Linux {
    Write-Host "Building $BINARY_NAME for Linux..." -ForegroundColor Cyan
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME-linux-amd64" .
    $env:GOARCH = "arm64"
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME-linux-aarch64" .
    Write-Host "Build complete: $BUILD_DIR/$BINARY_NAME-linux-amd64 and $BUILD_DIR/$BINARY_NAME-linux-aarch64" -ForegroundColor Green
}

function Build-Darwin {
    Write-Host "Building $BINARY_NAME for macOS..." -ForegroundColor Cyan
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    $env:GOOS = "darwin"
    $env:GOARCH = "amd64"
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME-darwin-amd64" .
    $env:GOARCH = "arm64"
    go build -ldflags="$LDFLAGS" -o "$BUILD_DIR/$BINARY_NAME-darwin-arm64" .
    Write-Host "Build complete: $BUILD_DIR/$BINARY_NAME-darwin-amd64 and $BUILD_DIR/$BINARY_NAME-darwin-arm64" -ForegroundColor Green
}

function Build-All {
    Build-Windows
    Build-Linux
    Build-Darwin
    Write-Host "All platform builds complete!" -ForegroundColor Green
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test ./... -v
}

function Run-Clean {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan
    if (Test-Path $BUILD_DIR) {
        Remove-Item -Recurse -Force $BUILD_DIR
    }
    if (Test-Path "coverage.out") {
        Remove-Item "coverage.out"
    }
    if (Test-Path "coverage.html") {
        Remove-Item "coverage.html"
    }
    Write-Host "Clean complete" -ForegroundColor Green
}

function Show-Help {
    Write-Host "Terminal Intelligence (TI) Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [target]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Targets:" -ForegroundColor Yellow
    Write-Host "  build            - Build for current platform (default)"
    Write-Host "  windows          - Build for Windows (amd64)"
    Write-Host "  linux            - Build for Linux (amd64 and arm64)"
    Write-Host "  darwin           - Build for macOS (amd64 and arm64)"
    Write-Host "  all              - Build for all platforms"
    Write-Host "  test             - Run all tests"
    Write-Host "  clean            - Remove build artifacts"
    Write-Host "  help             - Show this help message"
}

# Execute target
switch ($Target.ToLower()) {
    "build" { Build-Current }
    "windows" { Build-Windows }
    "linux" { Build-Linux }
    "darwin" { Build-Darwin }
    "all" { Build-All }
    "test" { Run-Tests }
    "clean" { Run-Clean }
    "help" { Show-Help }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Show-Help
        exit 1
    }
}
