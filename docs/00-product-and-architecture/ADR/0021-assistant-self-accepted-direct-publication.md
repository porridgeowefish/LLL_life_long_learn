# ADR-0021: Assistant Self-Accepted Direct Publication

Status: accepted
Owner: project maintainer
Date: 2026-09-13
Last reviewed: 2026-09-13
Source of truth: assistant-task output acceptance and publication boundary.
Supersedes: ADR-0020
Extends: ADR-0012

## Context

The previous dispatcher treated an external assistant's output as untrusted. It
checked result-manifest task constraints, per-file declarations, content hashes,
and SVG contents before promoting an output. It also kept a specialised retry
branch for artifact-only promotion failures.

LLL now delegates to a capable external assistant that performs its own output
acceptance. The duplicate Go acceptance layer caused valid work to remain in an
attempt directory and introduced a second, hard-to-explain task lifecycle.

## Decision

The assistant owns acceptance of its work. After the CLI writes a parseable
`result-manifest.json`, LLL publishes the asset candidates and every file below
each declared deliverable directory into the project asset store. `artifact.json`
contains display metadata and receives LLL provenance during publication; it no
longer needs a server-verified file list, byte count, or content hash.

LLL derives a presentation-only file index after copying the directory so the
reader can count files and recognize a Markdown entry point. This index does
not participate in acceptance and does not contain hashes.

LLL keeps only operational storage duties:

- create an isolated attempt workspace and capture the task context for the CLI;
- confine publication reads to that workspace and writes to the target project;
- atomically stage and publish artifact directories;
- version or merge core teaching assets, persist task state, and emit UI events.

LLL does not evaluate whether declared outputs satisfy a task type, whether all
core assets were mentioned, whether generated file metadata matches contents, or
whether SVG content meets a server-side policy. Failed filesystem publication is
terminal and reported as `publish-failed` or `partial-publish`; it is not
revalidated or retried automatically.

## Consequences

- A completed assistant directory is immediately available in `assets/generated`
  and the teacher's asset view after publication succeeds.
- Task records remain useful execution history, but no longer represent a second
  content-acceptance authority.
- Existing attempts with older hash fields remain readable; the fields are ignored.
- The bounded recovery rules and failure codes from ADR-0020 are retired.

## Rejected Alternatives

- Retain semantic validation while skipping only SHA-256 checks.
- Keep the artifact-only recovery branch after removing its validation predicate.
- Let the CLI write directly into the formal asset tree without an isolated
  workspace or atomic publication step.
