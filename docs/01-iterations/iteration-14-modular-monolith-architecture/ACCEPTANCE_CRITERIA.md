# Iteration 14 Acceptance Criteria

Status: automated acceptance complete; user-run native browser/visible-CLI acceptance pending
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: observable acceptance conditions for the modular-monolith refactor.

## Repository Navigation

- [x] A maintainer can locate the public entry point for every primary business
  capability from the repository map.
- [x] The application composition, HTTP transport, business modules, platform
  mechanics, and legacy compatibility code have distinct documented roots.
- [x] Frontend teacher, assistant-task, asset, source, project, preference, and
  legacy code each have a single documented feature entry point.

## Dependency Boundaries

- [x] An automated check rejects HTTP transport importing a module-private
  store, provider, executor, or filesystem implementation.
- [x] An automated check rejects one business module importing another
  module's private implementation.
- [x] An automated check rejects platform code importing a business module.
- [x] Only the application bootstrap composes concrete implementations across
  module and platform boundaries.
- [x] Introducing an intentional new dependency requires updating the checked
  architecture policy in the same change.

## Configuration

- [x] One typed loader applies defaults, local JSON, environment variables, and
  command-line overrides in the documented order.
- [x] Existing flat local configuration remains readable and resolves to the
  same effective values.
- [x] Loading legacy configuration does not rewrite the user's file.
- [x] Invalid types, unsupported enum values, and unsafe workspace paths fail
  with an actionable diagnostic.
- [x] Logged or returned diagnostics redact configured secrets.
- [x] A module receives only its own typed configuration slice.

## Behavior And Data Compatibility

- [x] Existing public HTTP paths, status behavior, JSON bodies, and SSE event
  shapes pass the frozen contract suite.
- [x] Current iteration-13 project fixtures remain readable without mutation.
- [x] Legacy Intro, Explain, Practice, Extend, Summary, and memory-era fixtures
  remain readable through the compatibility boundary.
- [x] No iteration-14 operation writes a retired legacy format.
- [x] Conversation, task, asset, source, project, and preference data each have
  one canonical write path.
- [x] A build produced before iteration 14 can still read product data written
  by iteration 14 because no persisted product schema changes.

## Quality And Reports

- [x] `check:fast`, `check`, and `check:full` have documented, non-overlapping
  purposes and return a non-zero status when a required check fails.
- [x] The complete gate runs Go tests and build, frontend lint/tests/build,
  architecture checks, and contract tests.
- [x] The release gate adds critical browser paths and Windows-native CLI smoke
  checks.
- [x] Test, coverage, and dependency reports are emitted below
  `.artifacts/quality/` and remain ignored by Git.
- [x] Coverage thresholds are based on the recorded baseline rather than an
  invented percentage.

## Windows Safety

- [ ] A real Windows smoke verifies a Chinese project path.
- [x] PowerShell-consumed generated scripts retain UTF-8 BOM where required.
- [x] A long prompt containing ASCII double quotes arrives complete at the
  native CLI boundary.
- [ ] Visible CLI execution and restart reconciliation retain their current
  observable behavior.

## Delivery

- [x] Every migration wave has a reviewable commit and recorded verification.
- [x] Temporary forwarding paths are absent from the final composition root.
- [x] All affected current documentation describes delivered reality and the
  delivery notes record residual risk.
