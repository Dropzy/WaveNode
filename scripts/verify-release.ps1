[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$cache = Join-Path $root '.cache'
New-Item -ItemType Directory -Force -Path $cache | Out-Null

function Invoke-Checked {
    param(
        [Parameter(Mandatory)]
        [string]$Command,

        [Parameter(ValueFromRemainingArguments)]
        [string[]]$Arguments
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE."
    }
}

Write-Host 'Testing backend...'
Push-Location (Join-Path $root 'Backend')
try {
    Invoke-Checked -Command go -Arguments @('test', './...')
    Invoke-Checked -Command go -Arguments @(
        'run',
        'golang.org/x/vuln/cmd/govulncheck@v1.3.0',
        './...'
    )
    Invoke-Checked -Command go -Arguments @(
        'build',
        '-o',
        (Join-Path $cache 'wavenode-release-check'),
        './cmd/server'
    )
}
finally {
    Pop-Location
}

Write-Host 'Running PostgreSQL integration tests...'
try {
    Invoke-Checked -Command docker -Arguments @(
        'compose',
        '-p',
        'wavenode-release-integration',
        '--project-directory',
        $root,
        '-f',
        (Join-Path $root 'docker-compose.test.yml'),
        'up',
        '--abort-on-container-exit',
        '--exit-code-from',
        'subsonic-integration-tests'
    )
}
finally {
    & docker compose -p wavenode-release-integration --project-directory $root -f (Join-Path $root 'docker-compose.test.yml') down --volumes --remove-orphans
}

Write-Host 'Checking frontend...'
Push-Location (Join-Path $root 'Frontend')
try {
    Invoke-Checked -Command npm -Arguments @('ci')
    Invoke-Checked -Command npm -Arguments @('audit', '--omit=dev', '--audit-level=high')
    Invoke-Checked -Command npm -Arguments @('run', 'lint', '--', '--max-warnings=0')
    Invoke-Checked -Command npm -Arguments @('run', 'build')
}
finally {
    Pop-Location
}

Write-Host 'Validating Docker Compose...'
$env:POSTGRES_PASSWORD = 'release-check-password'
$env:JWT_SECRET = 'release-check-secret-that-is-at-least-thirty-two-characters'
$env:MUSIC_PATH = $root
Invoke-Checked -Command docker -Arguments @(
    'compose',
    '--project-directory',
    $root,
    'config',
    '--quiet'
)

Write-Host 'Checking patch hygiene...'
Invoke-Checked -Command git -Arguments @('-C', $root, 'diff', '--check')

Write-Host 'Release verification passed.'
