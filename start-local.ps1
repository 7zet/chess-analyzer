$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$runtimeDir = Join-Path $projectRoot ".runtime"
$backendDir = Join-Path $projectRoot "backend"
$webDir = Join-Path $projectRoot "web"
$apiBinary = Join-Path $runtimeDir "royal-api.exe"
$pidFile = Join-Path $runtimeDir "processes.json"
$stockfish = Join-Path $projectRoot "tools\stockfish\stockfish\stockfish-windows-x86-64-universal.exe"

New-Item -ItemType Directory -Path $runtimeDir -Force | Out-Null

if (!(Test-Path -LiteralPath $stockfish)) {
    & (Join-Path $projectRoot "setup-stockfish.ps1")
}

if (Test-Path -LiteralPath $pidFile) {
    Write-Host "Royal Move allaqachon ishga tushgan bo'lishi mumkin. Avval .\stop-local.ps1 ni bajaring." -ForegroundColor Yellow
    exit 1
}

Write-Host "Go API yig'ilmoqda..." -ForegroundColor DarkGreen
Push-Location $backendDir
try { & go build -o $apiBinary ./cmd/api } finally { Pop-Location }

if (!(Test-Path -LiteralPath (Join-Path $webDir "node_modules"))) {
    Write-Host "Frontend paketlari o'rnatilmoqda..." -ForegroundColor DarkGreen
    Push-Location $webDir
    try { & npm install } finally { Pop-Location }
}

$api = Start-Process -FilePath $apiBinary -WorkingDirectory $backendDir -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput (Join-Path $runtimeDir "api.log") -RedirectStandardError (Join-Path $runtimeDir "api-error.log")

$node = (Get-Command node).Source
$next = Join-Path $webDir "node_modules\next\dist\bin\next"
$web = Start-Process -FilePath $node -ArgumentList @($next, "dev") -WorkingDirectory $webDir -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput (Join-Path $runtimeDir "web.log") -RedirectStandardError (Join-Path $runtimeDir "web-error.log")

@{ api = $api.Id; web = $web.Id } | ConvertTo-Json | Set-Content -LiteralPath $pidFile
Start-Sleep -Seconds 3

try {
    $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 5
    Write-Host "API: $($health.status)" -ForegroundColor Green
} catch {
    Write-Host "API ishga tushmadi. .runtime\api-error.log faylini tekshiring." -ForegroundColor Red
}

Write-Host "Royal Move: http://localhost:3000" -ForegroundColor Green
Write-Host "To'xtatish: .\stop-local.ps1" -ForegroundColor Gray
