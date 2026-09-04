# Iteration 14 User Stories

Status: implemented; native acceptance handoff pending
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: maintainer and learner value targeted by the modular-monolith refactor.

## Maintainer Stories

### US-14.1 — Read by capability

As a new maintainer, I want teacher, assistant, asset, source, project, and
preference behavior to have obvious code entry points so that I can follow one
use case without searching unrelated technical folders.

### US-14.2 — Change within a boundary

As a maintainer, I want module internals to remain private so that changing a
store, provider, or dispatcher does not silently couple unrelated modules.

### US-14.3 — Understand application composition

As a maintainer, I want one bootstrap location to show how configuration,
modules, adapters, routes, recovery, and shutdown are assembled.

### US-14.4 — Know who owns a file

As a maintainer, I want every canonical persisted data family to have one
writer so that recovery and compatibility rules cannot be bypassed.

### US-14.5 — Add a later iteration safely

As a maintainer, I want a repeatable module and test structure so that a later
iteration has a clear location for new behavior, contracts, fixtures, and
quality evidence.

### US-14.6 — Configure once

As an operator, I want one validated configuration system with documented
precedence and redacted diagnostics so that modules do not interpret the same
local file differently.

### US-14.7 — Run one quality gate

As a contributor, I want standard fast, complete, and release verification
commands so that local and CI results mean the same thing.

### US-14.8 — See actionable evidence

As a reviewer, I want test, coverage, and architecture reports in a predictable
ignored output directory so that failures can be investigated without adding
generated noise to Git.

## Learner Protection Stories

### US-14.9 — Preserve current behavior

As a learner, I want the architecture refactor to preserve the teacher,
assistant, asset, source, project, preference, and navigation behavior I already
use.

### US-14.10 — Preserve local data

As a learner with current or legacy projects, I want the new build to read my
existing files without deleting, rewriting, or silently replacing them.

### US-14.11 — Preserve native CLI behavior

As a Windows learner, I want visible Agent terminals, Chinese paths, complete
prompts, and restart recovery to continue working after process code moves.
