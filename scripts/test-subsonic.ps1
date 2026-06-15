$ErrorActionPreference = "Stop"
$projectName = "wavenode-subsonic-tests"
$repositoryRoot = Split-Path -Parent $PSScriptRoot

Push-Location $repositoryRoot
try {
  docker compose -p $projectName -f docker-compose.test.yml up `
    --build `
    --abort-on-container-exit `
    --exit-code-from subsonic-integration-tests

  $exitCode = $LASTEXITCODE
  docker compose -p $projectName -f docker-compose.test.yml down --volumes --remove-orphans
  exit $exitCode
}
finally {
  Pop-Location
}
