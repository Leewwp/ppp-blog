$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$composePath = Join-Path $repoRoot "validation\server\docker-compose.validation.yml"
$scriptPath = Join-Path $repoRoot "validation\scripts\validate-apis.sh"

$compose = Get-Content -Raw $composePath
$script = Get-Content -Raw $scriptPath
$errors = @()

if ($compose -notmatch "HALO_SECURITY_BASIC_AUTH_DISABLED=false") {
    $errors += "Validation stack must enable Basic Auth."
}

if ($compose -notmatch "SPRINGDOC_API_DOCS_ENABLED=true") {
    $errors += "Validation stack must enable springdoc API docs."
}

if ($script -notmatch "Accept: application/json") {
    $errors += "API validation script must request JSON responses explicitly."
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Host "FAIL: $_" }
    exit 1
}

Write-Host "API validation configuration checks passed."
