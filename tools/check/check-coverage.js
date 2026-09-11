// Compares measured coverage against the recorded iteration-14 baseline
// (tests/baseline.json). Fails when coverage drops more than one percentage
// point below the baseline. Usage:
//   node tools/check/check-coverage.js go <profile-file>
//   node tools/check/check-coverage.js frontend <coverage-final.json>
const fs = require('fs');

const [, , kind, path] = process.argv;
if (!kind || !path) {
  console.error('usage: node tools/check/check-coverage.js go|frontend <file>');
  process.exit(2);
}
const baseline = JSON.parse(fs.readFileSync('tests/baseline.json', 'utf8'));
const entry = baseline[kind];
if (!entry) {
  console.error(`baseline has no entry for ${kind}`);
  process.exit(2);
}

let covered = 0;
let total = 0;

if (kind === 'go') {
  const lines = fs.readFileSync(path, 'utf8').split('\n').slice(1);
  for (const line of lines) {
    const fields = line.trim().split(/\s+/);
    if (fields.length < 3) continue;
    const count = fields[1];
    const hit = fields[2];
    if (!/^\d+$/.test(count) || !/^\d+$/.test(hit)) continue;
    total += Number(count);
    covered += Number(hit);
  }
} else {
  const data = JSON.parse(fs.readFileSync(path, 'utf8'));
  for (const entry of Object.values(data)) {
    for (const count of Object.values(entry.s || {})) {
      total += 1;
      if (count > 0) covered += 1;
    }
  }
}

if (total === 0) {
  console.error(`no statements parsed from ${path}`);
  process.exit(2);
}
const pct = (covered / total) * 100;
const floor = entry.statementCoveragePct - 1.0;
const fmt = (v) => v.toFixed(2);
console.log(`${kind}: ${fmt(pct)}% (${covered}/${total}); baseline ${fmt(entry.statementCoveragePct)}%, floor ${fmt(floor)}%`);
if (pct < floor) {
  console.error(`FAIL: ${kind} coverage fell below the baseline floor`);
  process.exit(1);
}
console.log('coverage gate: PASS');
