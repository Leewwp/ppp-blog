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

if ($loadScript -match "\?\.|\?\[|\?\?") {
    $errors += "Load test must avoid optional chaining and nullish syntax because the pinned k6 runtime cannot parse it."
}

if ($stressScript -match "\?\.|\?\[|\?\?") {
    $errors += "Stress test must avoid optional chaining and nullish syntax because the pinned k6 runtime cannot parse it."
}

if ($loadScript -match "load-test-results\.json") {
    $errors += "Load test handleSummary must not write a relative results file because GitHub Actions already exports the summary and JSON outputs via mounted /results paths."
}

if ($stressScript -match "stress-test-results\.json") {
    $errors += "Stress test handleSummary must not write a relative results file because GitHub Actions already exports the summary and JSON outputs via mounted /results paths."
}

if ($loadScript -match "health check response time < 200ms") {
    $errors += "Load test must not hard-code a 200ms health-check assertion because the remote GitHub runner to Guangzhou path adds network latency that should be evaluated by global thresholds instead."
}

if ($loadScript -notmatch "http_req_duration" -or $loadScript -notmatch "http_req_failed") {
    $errors += "Load test must keep explicit latency and failure-rate thresholds."
}

if ($stressScript -match "thresholds\s*:") {
    $errors += "Stress test should report metrics without hard gating on k6 thresholds."
}

if ($validationWorkflow -notmatch "chmod 0777 validation-artifacts validation-artifacts/test-results") {
    $errors += "Validation workflow must make the local k6 results directory writable before mounting it into the container."
}

if ($validationWorkflow -notmatch '--user "\$\(id -u\):\$\(id -g\)"') {
    $errors += "Validation workflow must run k6 with the runner uid/gid so summary outputs can be written to the mounted results directory."
}

if ($validationWorkflow -match "steps\.stress-tests\.outcome == 'failure'") {
    $errors += "Validation workflow must not gate deployment on the informational stress test outcome."
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Host "FAIL: $_" }
    exit 1
}

Write-Host "Performance script contract checks passed."
