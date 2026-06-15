param(
  [int]$BackendPort = 8080,
  [int]$FrontendPort = 5173,
  [string]$Username = "live-test-admin",
  [string]$Password = "admin123"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$backupPath = Join-Path $root ".cache\wavenode-smoke-backup.zip"
$createdPlaylistId = $null
$createdUserId = $null

function Invoke-Api {
  param(
    [Parameter(Mandatory)][string]$Method,
    [Parameter(Mandatory)][string]$Path,
    [object]$Body,
    [hashtable]$Headers = @{}
  )

  $parameters = @{
    Method = $Method
    Uri = "http://127.0.0.1:$BackendPort/api$Path"
    Headers = $Headers
  }
  if ($null -ne $Body) {
    $parameters.ContentType = "application/json"
    $parameters.Body = $Body | ConvertTo-Json -Depth 8
  }
  Invoke-RestMethod @parameters
}

try {
  & (Join-Path $PSScriptRoot "start-live-test.ps1") -BackendPort $BackendPort -FrontendPort $FrontendPort -Username $Username -Password $Password

  $health = Invoke-RestMethod "http://127.0.0.1:$BackendPort/health"
  if ($health.status -ne "healthy" -or -not $health.version) {
    throw "Backend health or version check failed"
  }

  $login = Invoke-Api -Method Post -Path "/auth/login" -Body @{ username = $Username; password = $Password }
  $token = $login.data.token
  if (-not $token) {
    throw "Authentication did not return a token"
  }
  $auth = @{ Authorization = "Bearer $token" }

  $me = Invoke-Api -Method Get -Path "/auth/me" -Headers $auth
  if ($me.data.username -ne $Username) {
    throw "Current account check failed"
  }

  $music = Invoke-Api -Method Get -Path "/music" -Headers $auth
  $tracks = @($music.data)
  $playlist = Invoke-Api -Method Post -Path "/playlists" -Headers $auth -Body @{
    name = "WaveNode smoke test"
    description = "Created by the release smoke test"
    track_ids = @()
  }
  $createdPlaylistId = $playlist.data.id

  if ($tracks.Count -gt 0) {
    $track = $tracks[0]
    $searchTerm = [System.Uri]::EscapeDataString($track.title)
    $search = Invoke-Api -Method Get -Path "/music/search?q=$searchTerm" -Headers $auth
    if (@($search.data).Count -lt 1) {
      throw "Library search check failed"
    }

    $streamRequest = [System.Net.HttpWebRequest]::Create("http://127.0.0.1:$BackendPort/api/music/$($track.id)/stream")
    $streamRequest.Headers["Authorization"] = "Bearer $token"
    $streamRequest.AddRange(0, 1023)
    $streamResponse = $streamRequest.GetResponse()
    if ([int]$streamResponse.StatusCode -ne 206 -or -not $streamResponse.Headers["Content-Range"]) {
      throw "Audio range streaming check failed"
    }
    $streamResponse.Close()

    Invoke-Api -Method Post -Path "/playlists/$createdPlaylistId/tracks" -Headers $auth -Body @{ track_id = $track.id } | Out-Null
    $playlistTracks = Invoke-Api -Method Get -Path "/playlists/$createdPlaylistId/tracks" -Headers $auth
    if (@($playlistTracks.data).Count -ne 1) {
      throw "Playlist add-track check failed"
    }
    Invoke-Api -Method Delete -Path "/playlists/$createdPlaylistId/tracks/$($track.id)" -Headers $auth | Out-Null

    Invoke-Api -Method Post -Path "/liked-tracks/$($track.id)" -Headers $auth | Out-Null
    $liked = Invoke-Api -Method Get -Path "/liked-tracks" -Headers $auth
    if (@($liked.data | Where-Object { $_.id -eq $track.id }).Count -ne 1) {
      throw "Liked-track persistence check failed"
    }
    Invoke-Api -Method Delete -Path "/liked-tracks/$($track.id)" -Headers $auth | Out-Null
  }

  $status = Invoke-Api -Method Get -Path "/admin/system/status" -Headers $auth
  $diagnostics = Invoke-Api -Method Get -Path "/admin/library/diagnostics" -Headers $auth
  if (-not $status.data.version -or $null -eq $diagnostics.data.indexed_tracks) {
    throw "Administration diagnostics check failed"
  }

  $testUsername = "smoke-user-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
  $createdUser = Invoke-Api -Method Post -Path "/admin/users" -Headers $auth -Body @{
    username = $testUsername
    email = "$testUsername@example.com"
    password = "smoke-test-password"
    role = "user"
  }
  $createdUserId = $createdUser.data.id
  $userLogin = Invoke-Api -Method Post -Path "/auth/login" -Body @{ username = $testUsername; password = "smoke-test-password" }
  $userAuth = @{ Authorization = "Bearer $($userLogin.data.token)" }
  if ($createdPlaylistId) {
    try {
      Invoke-Api -Method Get -Path "/playlists/$createdPlaylistId" -Headers $userAuth | Out-Null
      throw "Regular user unexpectedly accessed another account's playlist"
    } catch {
      if ($_.Exception.Response.StatusCode.value__ -ne 404) {
        throw
      }
    }
  }
  try {
    Invoke-Api -Method Get -Path "/admin/stats" -Headers $userAuth | Out-Null
    throw "Regular user unexpectedly accessed an administration endpoint"
  } catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 403) {
      throw
    }
  }
  Invoke-Api -Method Put -Path "/auth/password" -Headers $userAuth -Body @{
    current_password = "smoke-test-password"
    new_password = "smoke-test-password-updated"
  } | Out-Null
  try {
    Invoke-Api -Method Get -Path "/auth/me" -Headers $userAuth | Out-Null
    throw "Password change did not invalidate the existing session"
  } catch {
    if ($_.Exception.Response.StatusCode.value__ -ne 401) {
      throw
    }
  }
  $userLogin = Invoke-Api -Method Post -Path "/auth/login" -Body @{
    username = $testUsername
    password = "smoke-test-password-updated"
  }
  if (-not $userLogin.data.token) {
    throw "Updated password login check failed"
  }

  Invoke-WebRequest -Uri "http://127.0.0.1:$BackendPort/api/admin/backup" -Headers $auth -OutFile $backupPath -UseBasicParsing
  if (-not (Test-Path $backupPath) -or (Get-Item $backupPath).Length -lt 100) {
    throw "Backup download check failed"
  }

  $restoreResult = & curl.exe --silent --show-error --fail `
    -H "Authorization: Bearer $token" `
    -F "backup=@$backupPath;type=application/zip" `
    "http://127.0.0.1:$BackendPort/api/admin/restore"
  if ($LASTEXITCODE -ne 0 -or ($restoreResult -join "") -notmatch '"success":true') {
    throw "Backup restore check failed"
  }
  $login = Invoke-Api -Method Post -Path "/auth/login" -Body @{ username = $Username; password = $Password }
  $token = $login.data.token
  $auth = @{ Authorization = "Bearer $token" }

  $frontend = Invoke-WebRequest "http://127.0.0.1:$FrontendPort" -UseBasicParsing
  if ($frontend.StatusCode -ne 200 -or $frontend.Content -notmatch "<title>WaveNode</title>") {
    throw "Frontend shell check failed"
  }

  Write-Host "Live smoke test passed."
} finally {
  if ($token) {
    $auth = @{ Authorization = "Bearer $token" }
    if ($createdPlaylistId) {
      try { Invoke-Api -Method Delete -Path "/playlists/$createdPlaylistId" -Headers $auth | Out-Null } catch {}
    }
    if ($createdUserId) {
      try { Invoke-Api -Method Delete -Path "/admin/users/$createdUserId" -Headers $auth | Out-Null } catch {}
    }
  }
  Remove-Item $backupPath -Force -ErrorAction SilentlyContinue
  & (Join-Path $PSScriptRoot "stop-live-test.ps1") -BackendPort $BackendPort -FrontendPort $FrontendPort
}
