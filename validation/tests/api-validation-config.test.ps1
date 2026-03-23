$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$workflowPath = Join-Path $repoRoot ".github\workflows\validation.yml"
$scriptPath = Join-Path $repoRoot "validation\scripts\validate-apis.sh"

$workflow = Get-Content -Raw $workflowPath
$script = Get-Content -Raw $scriptPath
$errors = @()

if ($workflow -notmatch "(?ms)api-validation:.*?Start Halo.*?env:\s*(?:[^\n]*\n)*?\s+HALO_SECURITY_BASIC_AUTH_DISABLED:\s*['""]?false['""]?") {
    $errors += "API validation workflow must enable Basic Auth before bootRun."
}

if ($workflow -notmatch "(?ms)api-validation:.*?Start Halo.*?env:\s*(?:[^\n]*\n)*?\s+SPRINGDOC_API_DOCS_ENABLED:\s*['""]?true['""]?") {
    $errors += "API validation workflow must enable springdoc API docs before bootRun."
}

if ($script -notmatch "Accept: application/json") {
    $errors += "API validation script must request JSON responses explicitly."
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Host "FAIL: $_" }
    exit 1
}

Write-Host "API validation configuration checks passed."
