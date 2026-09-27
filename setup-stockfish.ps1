$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$targetDir = Join-Path $projectRoot "tools\stockfish"
$binary = Join-Path $targetDir "stockfish\stockfish-windows-x86-64-universal.exe"

if (Test-Path -LiteralPath $binary) {
    Write-Host "Stockfish is already installed." -ForegroundColor Green
    exit 0
}

Write-Host "Downloading the official Stockfish 19 Windows release..." -ForegroundColor DarkGreen
$headers = @{ "User-Agent" = "RoyalMove-Setup" }
$release = Invoke-RestMethod -Uri "https://api.github.com/repos/official-stockfish/Stockfish/releases/tags/sf_19" -Headers $headers
$asset = $release.assets | Where-Object { $_.name -eq "stockfish-windows-x86-64-universal.zip" }
if (!$asset) { throw "The Stockfish Windows release asset was not found." }

New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
$archive = Join-Path $targetDir $asset.name
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $archive -UseBasicParsing
Expand-Archive -LiteralPath $archive -DestinationPath $targetDir -Force
Remove-Item -LiteralPath $archive -Force

if (!(Test-Path -LiteralPath $binary)) { throw "Stockfish installation did not produce the expected binary." }
Write-Host "Stockfish 19 installed successfully." -ForegroundColor Green
