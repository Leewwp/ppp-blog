$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$validationWorkflowPath = Join-Path $repoRoot ".github\workflows\validation.yml"
$deployWorkflowPath = Join-Path $repoRoot ".github\workflows\deploy.yml"
$setupValidationPath = Join-Path $repoRoot "validation\server\setup-validation.sh"
$readmePath = Join-Path $repoRoot "README.md"
$autoDeploymentPath = Join-Path $repoRoot "AUTO_DEPLOYMENT.md"

$validationWorkflow = Get-Content -Raw $validationWorkflowPath
$deployWorkflow = Get-Content -Raw $deployWorkflowPath
$setupValidation = Get-Content -Raw $setupValidationPath
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

if ($validationWorkflow -match "<<EOF") {
    $errors += "Validation workflow must not use inline heredocs inside YAML run/script blocks because they can break YAML indentation."
}

if ($validationWorkflow -match "git clone /opt/halo-blog") {
    $errors += "Validation workflow must not clone from /opt/halo-blog because that path can lag behind the validated GitHub commit."
}

if ($validationWorkflow -match 'git reset --hard "\$VALIDATED_SHA"') {
    $errors += "Validation workflow must not reset the server copy to VALIDATED_SHA when the server is not fetching directly from the GitHub repository."
}

if ($validationWorkflow -match "git remote prune origin" -or $validationWorkflow -match "git update-ref -d refs/remotes/origin/main") {
    $errors += "Validation workflow must not delete remote refs while preparing the validation instance."
}

if ($validationWorkflow -match "appleboy/scp-action") {
    $errors += "Validation workflow must not depend on scp-action for validation deployment because cross-border uploads are timing out."
}

if ($validationWorkflow -match 'git clone "\$VALIDATION_REPOSITORY"' -or $validationWorkflow -match 'git fetch --depth=1 origin "\$VALIDATED_SHA"' -or $validationWorkflow -match 'git checkout --force "\$VALIDATED_SHA"') {
    $errors += "Validation workflow must not depend on the server fetching the validated commit from GitHub during deployment."
}

if ($validationWorkflow -notmatch 'base64 -w0 validation/server/docker-compose\.validation\.yml' -or $validationWorkflow -notmatch 'base64 -w0 mysql-init/02-comment-tables\.sql') {
    $errors += "Validation workflow must package the validation stack assets from the runner before opening the SSH session."
}

if ($validationWorkflow -notmatch 'decode_to_file "\$VALIDATION_DOCKER_COMPOSE_B64" "\$VALIDATION_SERVER_PATH/validation/server/docker-compose\.validation\.yml"' -or $validationWorkflow -notmatch 'decode_to_file "\$VALIDATION_MYSQL_INIT_B64" "\$VALIDATION_SERVER_PATH/mysql-init/02-comment-tables\.sql"') {
    $errors += "Validation workflow must reconstruct the validation stack files on the server from the runner-provided payload."
}

if ($validationWorkflow -notmatch 'POST "\$VALIDATION_BASE_URL/system/setup"' -or $validationWorkflow -notmatch 'Accept: application/json') {
    $errors += "Validation workflow must initialize Halo with an explicit JSON Accept header."
}

if ($validationWorkflow -notmatch '\[ "\$status" != "204" \] && \[ "\$status" != "302" \] && \[ "\$status" != "409" \]') {
    $errors += "Validation workflow must treat 302 and 409 setup responses as idempotent initialization outcomes."
}

if ($validationWorkflow -notmatch "docker compose version" -or $validationWorkflow -notmatch "command -v docker-compose") {
    $errors += "Validation workflow must support both docker compose and docker-compose on the server."
}

if ($validationWorkflow -match 'find /tmp -name ''validation-source\.tgz''' -or $validationWorkflow -match 'tar -xzf "\$archive_path" -C "\$VALIDATION_SERVER_PATH"') {
    $errors += "Validation workflow must not unpack an uploaded validation tarball on the server."
}

if ($validationWorkflow -match 'for outcome in "\$\{\{ steps\.api-validation\.outcome \}\}" "\$\{\{ steps\.load-tests\.outcome \}\}" "\$\{\{ steps\.stress-tests\.outcome \}\}"') {
    $errors += "Validation failure gate must not treat skipped validation steps as failures."
}

if ($validationWorkflow -notmatch "steps\.api-validation\.outcome == 'failure'" -or $validationWorkflow -notmatch "steps\.load-tests\.outcome == 'failure'") {
    $errors += "Validation failure gate must fail when API validation or load tests actually fail."
}

if ($validationWorkflow -match "steps\.stress-tests\.outcome == 'failure'") {
    $errors += "Validation failure gate must not treat the stress test as a blocking deployment check."
}

if ($validationWorkflow -notmatch 'mkdir -p artifacts/validation-server-results') {
    $errors += "Validation summary must create the artifact directory before searching for result files."
}

if ($validationWorkflow -notmatch "actions/upload-artifact@v4" -or $validationWorkflow -notmatch "name: validation-server-results") {
    $errors += "Validation workflow must upload validation-server-results artifacts directly from the runner."
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

if ($deployWorkflow -match "git remote prune origin" -or $deployWorkflow -match "git update-ref -d refs/remotes/origin/main") {
    $errors += "Production deploy workflow must not delete remote refs before fetching the validated commit."
}

if ($setupValidation -notmatch "docker compose version" -or $setupValidation -notmatch "command -v docker-compose") {
    $errors += "Validation setup script must support both docker compose and docker-compose."
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
