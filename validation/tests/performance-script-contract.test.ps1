$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$loadScriptPath = Join-Path $repoRoot "validation\api-load.js"
$stressScriptPath = Join-Path $repoRoot "validation\stress-test.js"
$validationWorkflowPath = Join-Path $repoRoot ".github\workflows\validation.yml"

$loadScript = Get-Content -Raw $loadScriptPath
$stressScript = Get-Content -Raw $stressScriptPath
$validationWorkflow = Get-Content -Raw $validationWorkflowPath

$errors = @()

if ($loadScript -match "http\.post\(`\$\{CONSOLE_API\}/posts") {
    $errors += "Load test must avoid post-creation write traffic so CI gating stays stable."
}

if ($stressScript -match "http\.post\(`\$\{CONSOLE_API\}/posts") {
    $errors += "Stress test must avoid post-creation write traffic so results focus on server throughput instead of content mutation failures."
}

if ($loadScript -notmatch "http_req_duration" -or $loadScript -notmatch "http_req_failed") {
    $errors += "Load test must keep explicit latency and failure-rate thresholds."
}

if ($stressScript -match "thresholds\s*:") {
    $errors += "Stress test should report metrics without hard gating on k6 thresholds."
}

if ($validationWorkflow -match "steps\.stress-tests\.outcome == 'failure'") {
    $errors += "Validation workflow must not gate deployment on the informational stress test outcome."
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Host "FAIL: $_" }
    exit 1
}

Write-Host "Performance script contract checks passed."
