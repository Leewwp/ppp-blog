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
  const raw = fs.readFileSync(filePath, "utf8").replace(/^\uFEFF/, "");
  return JSON.parse(raw);
}

function round(value, digits = 2) {
  if (value === null || value === undefined || Number.isNaN(Number(value))) {
    return "n/a";
  }
  return Number(value).toFixed(digits);
}

function parseEventMetrics(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return {
      totalRequests: null,
      averageQps: null,
      peakQps: null,
      failureRate: null,
      avgLatency: null,
      p95Latency: null,
      p99Latency: null,
      maxVus: null,
    };
  }

  const lines = fs.readFileSync(filePath, "utf8").split(/\r?\n/).filter(Boolean);
  const buckets = new Map();
  const latencyValues = [];
  const failureValues = [];
  let totalRequests = 0;
  let maxVus = null;

  for (const line of lines) {
    let parsed;
    try {
      parsed = JSON.parse(line);
    } catch {
      continue;
    }

    if (parsed.type !== "Point") {
      continue;
    }

    const metricName = parsed.metric;
    const rawTime = parsed.data && parsed.data.time;
    const value = Number(parsed.data && parsed.data.value);
    if (!rawTime || Number.isNaN(value)) {
      continue;
    }

    if (metricName === "http_reqs" && value > 0) {
      const bucket = new Date(rawTime).toISOString().slice(0, 19);
      buckets.set(bucket, (buckets.get(bucket) || 0) + value);
      totalRequests += value;
      continue;
    }

    if (metricName === "http_req_duration" && value >= 0) {
      latencyValues.push(value);
      continue;
    }

    if (metricName === "http_req_failed" && value >= 0) {
      failureValues.push(value);
      continue;
    }

    if ((metricName === "vus" || metricName === "vus_max") && value >= 0) {
      maxVus = maxVus === null ? value : Math.max(maxVus, value);
    }
  }

  if (buckets.size === 0 && latencyValues.length === 0 && failureValues.length === 0 && maxVus === null) {
    return {
      totalRequests: null,
      averageQps: null,
      peakQps: null,
      failureRate: null,
      avgLatency: null,
      p95Latency: null,
      p99Latency: null,
      maxVus: null,
    };
  }

  latencyValues.sort((a, b) => a - b);
  const peakQps = buckets.size > 0 ? Math.max(...buckets.values()) : null;
  const averageQps = buckets.size > 0 ? totalRequests / buckets.size : null;
  const failureRate = failureValues.length > 0
    ? failureValues.reduce((sum, current) => sum + current, 0) / failureValues.length
    : null;
  const avgLatency = latencyValues.length > 0
    ? latencyValues.reduce((sum, current) => sum + current, 0) / latencyValues.length
    : null;

  return {
    totalRequests: totalRequests || null,
    averageQps,
    peakQps,
    failureRate,
    avgLatency,
    p95Latency: percentile(latencyValues, 0.95),
    p99Latency: percentile(latencyValues, 0.99),
    maxVus,
  };
}

function metric(summary, metricName, field) {
  if (!summary || !summary.metrics || !summary.metrics[metricName]) {
    return null;
  }
  const values = summary.metrics[metricName].values || {};
  return values[field] ?? null;
}

function percentile(sortedValues, ratio) {
  if (!sortedValues || sortedValues.length === 0) {
    return null;
  }

  const index = Math.ceil(sortedValues.length * ratio) - 1;
  const safeIndex = Math.max(0, Math.min(sortedValues.length - 1, index));
  return sortedValues[safeIndex];
}

function firstDefined(...values) {
  for (const value of values) {
    if (value !== null && value !== undefined) {
      return value;
    }
  }
  return null;
}

function renderScenario(name, summary, events) {
  const totalRequests = firstDefined(
    metric(summary, "http_reqs", "count"),
    events.totalRequests,
  );
  const failureRate = firstDefined(
    metric(summary, "http_req_failed", "rate"),
    events.failureRate,
    0,
  );
  const avgLatency = firstDefined(
    metric(summary, "http_req_duration", "avg"),
    events.avgLatency,
  );
  const p95Latency = firstDefined(
    metric(summary, "http_req_duration", "p(95)"),
    events.p95Latency,
  );
  const p99Latency = firstDefined(
    metric(summary, "http_req_duration", "p(99)"),
    events.p99Latency,
  );
  const maxVus = firstDefined(
    metric(summary, "vus_max", "max"),
    metric(summary, "vus_max", "value"),
    metric(summary, "vus", "max"),
    metric(summary, "vus", "value"),
    events.maxVus,
  );

  if (!summary && totalRequests === null && events.averageQps === null && avgLatency === null && maxVus === null) {
    return `## ${name}\n\nResult files are unavailable.\n`;
  }

  return [
    `## ${name}`,
    "",
    "| Metric | Value |",
    "| --- | --- |",
    `| Total Requests | ${round(totalRequests, 0)} |`,
    `| Average QPS | ${round(events.averageQps)} |`,
    `| Peak QPS | ${round(events.peakQps)} |`,
    `| Failure Rate | ${round(failureRate * 100)}% |`,
    `| Avg Latency | ${round(avgLatency)} ms |`,
    `| P95 Latency | ${round(p95Latency)} ms |`,
    `| P99 Latency | ${round(p99Latency)} ms |`,
    `| Max VUs | ${round(maxVus, 0)} |`,
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
