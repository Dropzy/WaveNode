param(
  [int]$BackendPort = 8080,
  [int]$FrontendPort = 5173
)

$ErrorActionPreference = "SilentlyContinue"

$root = Split-Path -Parent $PSScriptRoot
$cacheDir = Join-Path $root ".cache\live-test"
$cmdExe = (Get-Command cmd.exe).Source

foreach ($name in @("backend", "frontend")) {
  $pidFile = Join-Path $cacheDir "$name.pid"
  if (Test-Path $pidFile) {
    $processId = Get-Content $pidFile
    if ($processId) {
      $process = Get-Process -Id ([int]$processId) -ErrorAction SilentlyContinue
      if ($process -and $process.ProcessName -in @("cmd", "powershell", "go", "node", "music-server")) {
        & $cmdExe /c "taskkill /PID $processId /T /F >nul 2>&1"
        Write-Host "Stopped $name process $processId"
      }
    }
    Remove-Item $pidFile -Force
  }
}

foreach ($port in @($BackendPort, $FrontendPort)) {
  $connections = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne $PID) {
      & $cmdExe /c "taskkill /PID $($connection.OwningProcess) /T /F >nul 2>&1"
    }
  }
}

$workspaceCachePrefix = Join-Path $root ".cache\go-build"
Get-Process -Name "server" -ErrorAction SilentlyContinue | ForEach-Object {
  if ($_.Path -and $_.Path.StartsWith($workspaceCachePrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
  }
}
