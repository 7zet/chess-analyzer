$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$pidFile = Join-Path $projectRoot ".runtime\processes.json"

if (!(Test-Path -LiteralPath $pidFile)) {
    Write-Host "Royal Move lokal processlari topilmadi."
    exit 0
}

$processes = Get-Content -LiteralPath $pidFile | ConvertFrom-Json

function Stop-ProcessTree([int]$rootId) {
    $children = Get-CimInstance Win32_Process -Filter "ParentProcessId=$rootId" -ErrorAction SilentlyContinue
    foreach ($child in $children) { Stop-ProcessTree $child.ProcessId }
    Stop-Process -Id $rootId -ErrorAction SilentlyContinue
}

foreach ($processId in @($processes.api, $processes.web)) {
    if ($processId) { Stop-ProcessTree $processId }
}
Remove-Item -LiteralPath $pidFile -Force
Write-Host "Royal Move to'xtatildi." -ForegroundColor Green
