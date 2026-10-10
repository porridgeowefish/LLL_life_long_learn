// Prevent forced adds of local runtime state and generated artifacts. Git lets
// callers stage an ignored file with `git add -f`; this check makes that
// exception visible in both local quality gates and CI.
const { spawnSync } = require('child_process');

function git(args, options = {}) {
  const result = spawnSync('git', args, {
    cwd: process.cwd(),
    encoding: 'utf8',
    ...options,
  });
  if (result.error) throw result.error;
  return result;
}

// -c queries the Git index and -i applies standard ignore rules to those
// indexed paths. It catches a file staged with `git add -f` in one Git call;
// checking each path with check-ignore is prohibitively slow on Windows.
const tracked = git(['ls-files', '-z']);
const ignoredTracked = git(['ls-files', '-ci', '--exclude-standard', '-z']);
if (tracked.status !== 0 || ignoredTracked.status !== 0) {
  console.error(tracked.stderr || ignoredTracked.stderr || 'could not inspect the Git index');
  process.exit(1);
}

// Engineering docs and tests became local-only in v0.5.0 (see .gitignore).
// Files tracked before that switch may stay tracked and keep being updated.
const localOnlyLegacy = [
  /^docs\//,
  /^tests\//,
  /^frontend\/e2e\//,
  /_test\.go$/,
  /\.(?:test|spec)\.(?:js|ts|tsx)$/,
];
const violations = ignoredTracked.stdout
  .split('\0')
  .filter(Boolean)
  .filter(file => !localOnlyLegacy.some(pattern => pattern.test(file)));

if (violations.length > 0) {
  console.error('Tracked paths must not match .gitignore:');
  for (const file of violations) console.error(`- ${file}`);
  console.error('Remove the file from Git tracking or make it an intentional, documented repository asset.');
  process.exit(1);
}

console.log(`repository hygiene: PASS (${tracked.stdout.split('\0').filter(Boolean).length} tracked paths checked)`);
