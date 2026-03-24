#!/usr/bin/env node

const fs = require("fs");

function parseArgs(argv) {
  const args = {};
  for (let i = 2; i < argv.length; i += 2) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key || !key.startsWith("--") || value === undefined) {
      continue;
    }
    args[key.slice(2)] = value;
  }
  return args;
}

function readJson(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return null;
  }
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function round(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(Number(value))) {
    return "n/a";
  }
  return Number(value).toFixed(digits);
}

function parseEventMetrics(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return { averageQps: null, peakQps: null };
  }

  const lines = fs.readFileSync(filePath, "utf8").split(/\r?\n/).filter(Boolean);
  const buckets = new Map();
  let totalRequests = 0;

  for (const line of lines) {
    let parsed;
    try {
      parsed = JSON.parse(line);
    } catch {
      continue;
    }

    if (parsed.metric !== "http_reqs" || parsed.type !== "Point") {
      continue;
    }

    const rawTime = parsed.data && parsed.data.time;
    const value = Number(parsed.data && parsed.data.value) || 0;
    if (!rawTime || value <= 0) {
      continue;
    }

    const bucket = new Date(rawTime).toISOString().slice(0, 19);
    buckets.set(bucket, (buckets.get(bucket) || 0) + value);
    totalRequests += value;
  }

  if (buckets.size === 0) {
    return { averageQps: null, peakQps: null };
  }

  const peakQps = Math.max(...buckets.values());
  const averageQps = totalRequests / buckets.size;
  return { averageQps, peakQps };
}

function metric(summary, metricName, field) {
  if (!summary || !summary.metrics || !summary.metrics[metricName]) {
    return null;
  }
  const values = summary.metrics[metricName].values || {};
  return values[field] ?? null;
}

function renderScenario(name, summary, events) {
  if (!summary) {
    return `## ${name}\n\nResult files are unavailable.\n`;
  }

  return [
    `## ${name}`,
    "",
    "| Metric | Value |",
    "| --- | --- |",
    `| Total Requests | ${round(metric(summary, "http_reqs", "count"), 0)} |`,
    `| Average QPS | ${round(events.averageQps)} |`,
    `| Peak QPS | ${round(events.peakQps)} |`,
    `| Failure Rate | ${round((metric(summary, "http_req_failed", "rate") || 0) * 100)}% |`,
    `| Avg Latency | ${round(metric(summary, "http_req_duration", "avg"))} ms |`,
    `| P95 Latency | ${round(metric(summary, "http_req_duration", "p(95)"))} ms |`,
    `| P99 Latency | ${round(metric(summary, "http_req_duration", "p(99)"))} ms |`,
    `| Max VUs | ${round(metric(summary, "vus_max", "max"), 0)} |`,
    "",
  ].join("\n");
}

function renderApiValidation(summary) {
  if (!summary) {
    return "## API Validation\n\nResult files are unavailable.\n";
  }

  const passed = summary.passed ?? 0;
  const failed = summary.failed ?? 0;
  return [
    "## API Validation",
    "",
    "| Metric | Value |",
    "| --- | --- |",
    `| Passed Checks | ${passed} |`,
    `| Failed Checks | ${failed} |`,
    `| Overall Result | ${failed === 0 ? "pass" : "fail"} |`,
    "",
  ].join("\n");
}

const args = parseArgs(process.argv);
const apiSummary = readJson(args["api-summary"]);
const loadSummary = readJson(args["load-summary"]);
const stressSummary = readJson(args["stress-summary"]);
const loadEvents = parseEventMetrics(args["load-events"]);
const stressEvents = parseEventMetrics(args["stress-events"]);

const sections = [
  "## Remote Validation Metrics",
  "",
  renderApiValidation(apiSummary),
  renderScenario("Load Test", loadSummary, loadEvents),
  renderScenario("Stress Test", stressSummary, stressEvents),
];

process.stdout.write(sections.join("\n"));
