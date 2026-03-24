$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$validationWorkflowPath = Join-Path $repoRoot ".github\workflows\validation.yml"
$deployWorkflowPath = Join-Path $repoRoot ".github\workflows\deploy.yml"
$readmePath = Join-Path $repoRoot "README.md"
$autoDeploymentPath = Join-Path $repoRoot "AUTO_DEPLOYMENT.md"

$validationWorkflow = Get-Content -Raw $validationWorkflowPath
$deployWorkflow = Get-Content -Raw $deployWorkflowPath
$readme = Get-Content -Raw $readmePath
$autoDeployment = Get-Content -Raw $autoDeploymentPath

$errors = @()

if ($validationWorkflow -notmatch "VALIDATION_BASE_URL") {
    $errors += "Validation workflow must target a configurable remote validation base URL."
}

if ($validationWorkflow -notmatch "18090") {
    $errors += "Validation workflow must reference the dedicated validation port 18090."
}

if ($validationWorkflow -notmatch "GITHUB_STEP_SUMMARY") {
    $errors += "Validation workflow must publish a GitHub Actions summary."
}

if ($validationWorkflow -notmatch "appleboy/ssh-action") {
    $errors += "Validation workflow must deploy or manage the remote validation instance over SSH."
}

if ($deployWorkflow -notmatch "workflow_run") {
    $errors += "Production deploy workflow must be triggered by workflow_run."
}

if ($deployWorkflow -notmatch "Halo Validation Pipeline") {
    $errors += "Production deploy workflow must be gated by the Halo Validation Pipeline workflow."
}

if ($deployWorkflow -notmatch "github\.event\.workflow_run\.conclusion == 'success'") {
    $errors += "Production deploy workflow must only run when validation completes successfully."
}

if ($readme -notmatch "/opt/halo-validation" -and $autoDeployment -notmatch "/opt/halo-validation") {
    $errors += "Deployment docs must describe the /opt/halo-validation server path."
}

if ($readme -notmatch "18090" -and $autoDeployment -notmatch "18090") {
    $errors += "Deployment docs must describe validation port 18090."
}

if ($errors.Count -gt 0) {
    $errors | ForEach-Object { Write-Host "FAIL: $_" }
    exit 1
}

Write-Host "Pipeline contract checks passed."
