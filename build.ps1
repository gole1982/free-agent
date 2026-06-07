<#
.SYNOPSIS
    Build script for free-agent project
.DESCRIPTION
    Unified build script for compiling the free-agent project on Windows
#>

param(
    [Parameter(Mandatory=$false)]
    [string]$Target = "build"
)

$ProjectName = "free-agent"
$OutputDir = "bin"
$MainPath = "cmd/free-agent/main.go"

function Build-Main {
    Write-Host "Building $ProjectName..." -ForegroundColor Cyan
    
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir | Out-Null
    }
    
    go build -o "$OutputDir\$ProjectName.exe" $MainPath
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build completed: $OutputDir\$ProjectName.exe" -ForegroundColor Green
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
}

function Clean-Build {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan
    
    if (Test-Path "$OutputDir\*.exe") {
        Remove-Item "$OutputDir\*.exe" -Force
    }
    
    if (Test-Path "*.exe") {
        Remove-Item "*.exe" -Force
    }
    
    Write-Host "Clean completed" -ForegroundColor Green
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test ./internal/...
}

function Run-App {
    Build-Main
    Write-Host "Running $ProjectName..." -ForegroundColor Cyan
    & ".\$OutputDir\$ProjectName.exe"
}

switch ($Target.ToLower()) {
    "build"      { Build-Main }
    "clean"      { Clean-Build }
    "test"       { Run-Tests }
    "run"        { Run-App }
    default      {
        Write-Host "Unknown target: $Target" -ForegroundColor Yellow
        Write-Host "Available targets: build, clean, test, run"
        exit 1
    }
}
