param(
  [int]$BackendPort = 8080,
  [int]$FrontendPort = 5173,
  [string]$Username = "live-test-admin",
  [string]$Password = "admin123"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$cacheDir = Join-Path $root ".cache\live-test"
$goCache = Join-Path $root ".cache\go-build"
$backendDir = Join-Path $root "Backend"
$frontendDir = Join-Path $root "Frontend"
$cmdExe = (Get-Command cmd.exe).Source
$goExe = (Get-Command go).Source
$nodeExe = (Get-Command node).Source

New-Item -ItemType Directory -Force -Path $cacheDir, $goCache | Out-Null

function Normalize-ProcessPath {
  $environment = [System.Environment]::GetEnvironmentVariables()
  $pathValue = $environment["Path"]
  if (-not $pathValue) {
    $pathValue = $environment["PATH"]
  }

  [System.Environment]::SetEnvironmentVariable("PATH", $null, [System.EnvironmentVariableTarget]::Process)
  [System.Environment]::SetEnvironmentVariable("Path", $pathValue, [System.EnvironmentVariableTarget]::Process)
}

function Stop-ExistingProcess {
  param([string]$PidFile)

  if (Test-Path $PidFile) {
    $processId = Get-Content $PidFile -ErrorAction SilentlyContinue
    if ($processId) {
      $process = Get-Process -Id ([int]$processId) -ErrorAction SilentlyContinue
      if ($process -and $process.ProcessName -in @("cmd", "powershell", "go", "node", "music-server")) {
        & $cmdExe /c "taskkill /PID $processId /T /F >nul 2>&1"
      }
    }
    Remove-Item $PidFile -Force -ErrorAction SilentlyContinue
  }
}

function Stop-PortOwner {
  param([int]$Port)

  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  foreach ($connection in $connections) {
    if ($connection.OwningProcess -and $connection.OwningProcess -ne $PID) {
      & $cmdExe /c "taskkill /PID $($connection.OwningProcess) /T /F >nul 2>&1"
    }
  }
}

function Stop-WorkspaceServer {
  $workspaceCachePrefix = (Join-Path $root ".cache\go-build")
  Get-Process -Name "server" -ErrorAction SilentlyContinue | ForEach-Object {
    if ($_.Path -and $_.Path.StartsWith($workspaceCachePrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
      Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
  }
}

Normalize-ProcessPath

Stop-ExistingProcess (Join-Path $cacheDir "backend.pid")
Stop-ExistingProcess (Join-Path $cacheDir "frontend.pid")
Stop-WorkspaceServer
Stop-PortOwner $BackendPort
Stop-PortOwner $FrontendPort

$backendOut = Join-Path $cacheDir "backend.out.log"
$backendErr = Join-Path $cacheDir "backend.err.log"

$backendEnv = @{
  GOCACHE = $goCache
  SERVER_PORT = "$BackendPort"
  DATABASE_URL = "postgres://postgres:password@localhost:5432/music_server?sslmode=disable"
  DEV_SEED_TEST_USER = "true"
  DEV_TEST_USERNAME = $Username
  DEV_TEST_PASSWORD = $Password
  DEV_TEST_EMAIL = "$Username@example.com"
  DEV_TEST_ROLE = "admin"
}

$backendCommand = "set `"GOCACHE=$($backendEnv.GOCACHE)`" && set `"SERVER_PORT=$($backendEnv.SERVER_PORT)`" && set `"DATABASE_URL=$($backendEnv.DATABASE_URL)`" && set `"DEV_SEED_TEST_USER=$($backendEnv.DEV_SEED_TEST_USER)`" && set `"DEV_TEST_USERNAME=$($backendEnv.DEV_TEST_USERNAME)`" && set `"DEV_TEST_PASSWORD=$($backendEnv.DEV_TEST_PASSWORD)`" && set `"DEV_TEST_EMAIL=$($backendEnv.DEV_TEST_EMAIL)`" && set `"DEV_TEST_ROLE=$($backendEnv.DEV_TEST_ROLE)`" && `"$goExe`" run .\cmd\server 1> `"$backendOut`" 2> `"$backendErr`""

$backend = Start-Process $cmdExe -ArgumentList "/c", $backendCommand -WorkingDirectory $backendDir -WindowStyle Hidden -PassThru
$backend.Id | Set-Content (Join-Path $cacheDir "backend.pid")

$frontendCommand = "set `"VITE_BACKEND_PORT=$BackendPort`" && `"$nodeExe`" .\node_modules\vite\bin\vite.js --host 127.0.0.1 --port $FrontendPort --logLevel silent"
$frontend = Start-Process $cmdExe -ArgumentList "/c", $frontendCommand -WorkingDirectory $frontendDir -WindowStyle Hidden -PassThru
$frontend.Id | Set-Content (Join-Path $cacheDir "frontend.pid")

function Wait-ForUrl {
  param(
    [string]$Url,
    [int]$TimeoutSeconds = 90
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2
      if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
        return
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  }

  throw "Timed out waiting for $Url"
}

try {
  Wait-ForUrl "http://127.0.0.1:$BackendPort/health"
  Wait-ForUrl "http://127.0.0.1:$FrontendPort"

  $loginBody = @{
    username = $Username
    password = $Password
  } | ConvertTo-Json

  $login = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$BackendPort/api/auth/login" -ContentType "application/json" -Body $loginBody
  if (-not $login.success) {
    throw "Live test login failed"
  }
} catch {
  & (Join-Path $PSScriptRoot "stop-live-test.ps1") -BackendPort $BackendPort -FrontendPort $FrontendPort
  throw "$($_.Exception.Message). Check $backendErr for backend startup details."
}

Write-Host "Live test stack is ready."
Write-Host "Frontend: http://127.0.0.1:$FrontendPort"
Write-Host "Backend:  http://127.0.0.1:$BackendPort"
Write-Host "Login:    $Username / $Password"
Write-Host "Logs:     $cacheDir"
