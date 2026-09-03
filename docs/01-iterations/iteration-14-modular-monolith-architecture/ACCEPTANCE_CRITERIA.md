# Iteration 14 Acceptance Criteria

Status: approved design; implementation not started
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: observable acceptance conditions for the modular-monolith refactor.

## Repository Navigation

- [ ] A maintainer can locate the public entry point for every primary business
  capability from the repository map.
- [ ] The application composition, HTTP transport, business modules, platform
  mechanics, and legacy compatibility code have distinct documented roots.
- [ ] Frontend teacher, assistant-task, asset, source, project, preference, and
  legacy code each have a single documented feature entry point.

## Dependency Boundaries

- [ ] An automated check rejects HTTP transport importing a module-private
  store, provider, executor, or filesystem implementation.
- [ ] An automated check rejects one business module importing another
  module's private implementation.
- [ ] An automated check rejects platform code importing a business module.
- [ ] Only the application bootstrap composes concrete implementations across
  module and platform boundaries.
- [ ] Introducing an intentional new dependency requires updating the checked
  architecture policy in the same change.

## Configuration

- [ ] One typed loader applies defaults, local JSON, environment variables, and
  command-line overrides in the documented order.
- [ ] Existing flat local configuration remains readable and resolves to the
  same effective values.
- [ ] Loading legacy configuration does not rewrite the user's file.
- [ ] Invalid types, unsupported enum values, and unsafe workspace paths fail
  with an actionable diagnostic.
- [ ] Logged or returned diagnostics redact configured secrets.
- [ ] A module receives only its own typed configuration slice.

## Behavior And Data Compatibility

- [ ] Existing public HTTP paths, status behavior, JSON bodies, and SSE event
  shapes pass the frozen contract suite.
- [ ] Current iteration-13 project fixtures remain readable without mutation.
- [ ] Legacy Intro, Explain, Practice, Extend, Summary, and memory-era fixtures
  remain readable through the compatibility boundary.
- [ ] No iteration-14 operation writes a retired legacy format.
- [ ] Conversation, task, asset, source, project, and preference data each have
  one canonical write path.
- [ ] A build produced before iteration 14 can still read product data written
  by iteration 14 because no persisted product schema changes.

## Quality And Reports

- [ ] `check:fast`, `check`, and `check:full` have documented, non-overlapping
  purposes and return a non-zero status when a required check fails.
- [ ] The complete gate runs Go tests and build, frontend lint/tests/build,
  architecture checks, and contract tests.
- [ ] The release gate adds critical browser paths and Windows-native CLI smoke
  checks.
- [ ] Test, coverage, and dependency reports are emitted below
  `.artifacts/quality/` and remain ignored by Git.
- [ ] Coverage thresholds are based on the recorded baseline rather than an
  invented percentage.

## Windows Safety

- [ ] A real Windows smoke verifies a Chinese project path.
- [ ] PowerShell-consumed generated scripts retain UTF-8 BOM where required.
- [ ] A long prompt containing ASCII double quotes arrives complete at the
  native CLI boundary.
- [ ] Visible CLI execution and restart reconciliation retain their current
  observable behavior.

## Delivery

- [ ] Every migration wave has a reviewable commit and recorded verification.
- [ ] Temporary forwarding paths are absent from the final composition root.
- [ ] All affected current documentation describes delivered reality and the
  delivery notes record residual risk.

