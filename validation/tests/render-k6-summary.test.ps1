$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$renderScriptPath = Join-Path $repoRoot "validation\scripts\render-k6-summary.js"
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("render-k6-summary-test-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
    $apiSummaryPath = Join-Path $tempDir "api-validation-summary.json"
    $loadSummaryPath = Join-Path $tempDir "load-test-summary.json"
    $stressSummaryPath = Join-Path $tempDir "stress-test-summary.json"
    $loadEventsPath = Join-Path $tempDir "load-test-events.json"
    $stressEventsPath = Join-Path $tempDir "stress-test-events.json"

    [System.IO.File]::WriteAllText($apiSummaryPath, '{"passed":7,"failed":0}', $utf8NoBom)
    [System.IO.File]::WriteAllText($loadSummaryPath, '{"metrics":{}}', $utf8NoBom)
    [System.IO.File]::WriteAllText($stressSummaryPath, '{"metrics":{}}', $utf8NoBom)

    [System.IO.File]::WriteAllLines($loadEventsPath, @(
        '{"metric":"http_reqs","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":1}}',
        '{"metric":"http_reqs","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":1}}',
        '{"metric":"http_reqs","type":"Point","data":{"time":"2026-03-24T00:00:01Z","value":1}}',
        '{"metric":"http_req_duration","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":123.45}}',
        '{"metric":"http_req_failed","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":0}}',
        '{"metric":"http_req_failed","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":1}}',
        '{"metric":"http_req_failed","type":"Point","data":{"time":"2026-03-24T00:00:01Z","value":0}}',
        '{"metric":"vus","type":"Point","data":{"time":"2026-03-24T00:00:01Z","value":7}}'
    ), $utf8NoBom)

    [System.IO.File]::WriteAllLines($stressEventsPath, @(
        '{"metric":"http_reqs","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":2}}',
        '{"metric":"http_req_duration","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":250.00}}',
        '{"metric":"http_req_failed","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":0}}',
        '{"metric":"vus_max","type":"Point","data":{"time":"2026-03-24T00:00:00Z","value":11}}'
    ), $utf8NoBom)

    $output = node $renderScriptPath `
        --api-summary $apiSummaryPath `
        --load-summary $loadSummaryPath `
        --load-events $loadEventsPath `
        --stress-summary $stressSummaryPath `
        --stress-events $stressEventsPath

    if ($LASTEXITCODE -ne 0) {
        throw "render-k6-summary.js exited with code $LASTEXITCODE"
    }

    if (-not $output.Contains('| Total Requests | 3 |') -or -not $output.Contains('| Average QPS | 1.50 |') -or -not $output.Contains('| Peak QPS | 2.00 |')) {
        throw "Expected load test request metrics to be derived from event data.`n$output"
    }

    if (-not $output.Contains('| Failure Rate | 33.33% |') -or -not $output.Contains('| Avg Latency | 123.45 ms |') -or -not $output.Contains('| P95 Latency | 123.45 ms |')) {
        throw "Expected load test latency and failure metrics to fall back to event data.`n$output"
    }

    if (-not $output.Contains('| Max VUs | 7 |') -or -not $output.Contains('## Stress Test') -or -not $output.Contains('| Total Requests | 2 |')) {
        throw "Expected render output to include stress test fallback metrics.`n$output"
    }
}
finally {
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
}

Write-Host "Render k6 summary checks passed."
