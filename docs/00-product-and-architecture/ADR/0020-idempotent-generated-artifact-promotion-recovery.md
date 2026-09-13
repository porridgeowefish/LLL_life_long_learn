# ADR-0020: Idempotent Generated Artifact Promotion Recovery

Status: superseded by ADR-0021
Owner: project maintainer
Date: 2026-09-13
Last reviewed: 2026-09-13
Source of truth: recovery boundary for a validated assistant deliverable that did not reach the project asset store.
Supersedes: ADR-0012's rejected general automatic Agent retry only for the narrow sealed artifact-promotion recovery defined here.
Superseded by: ADR-0021
Extends: ADR-0012 and ADR-0015

## Context

An assistant can finish its visible CLI run and write a valid, hashed
`result-manifest.json` plus a generated deliverable, yet fail while Go promotes
that deliverable from the sealed attempt workspace into `assets/generated/`.
The learner then has a completed file under the attempt directory but no
generated artifact API result or asset-page card. Treating it as a normal task
failure strands an otherwise valid learner-owned output.

Re-running the CLI would create a second model execution and may produce a
different document. Retrying every failed task would also violate ADR-0012's
learner-controlled task lifecycle and could repeat asset merges or source
writes.

## Decision

The dispatcher may perform one idempotent promotion recovery on its normal
reconciliation loop when every condition below is true:

- the task is `produce-material` and its failure is `commit-failed`;
- the sealed manifest and every declared output still validate;
- it declares one or more generated deliverables, no source update, and all
  three core assets (`intro`, `body`, `practice`) as `unchanged`.

The recovery repeats only Go's validated artifact-promotion step against the
same task id, run id, sealed inputs, and attempt files. It never reopens the
CLI, calls a model, starts another task, merges a core asset candidate, or
changes the learner's conversation cutoff. A successful recovery marks the
same task `succeeded` and emits the existing task and generated-artifact events.

If that recovery fails again, the task becomes `artifact-recovery-failed` and
is not retried automatically. The existing task failure `suggestion` names the
output class that was not committed, so the learner and maintainer can inspect
the saved attempt without exposing raw filesystem errors.

If the durable generated-artifact record proves that promotion nevertheless
completed while the task still records `artifact-recovery-failed`, reconciliation
repairs that stale task projection to `succeeded`. This convergence step does
not write a file or retry promotion.

No new file schema or HTTP field is introduced: the recovery uses the existing
task, attempt, result manifest, commit journal, and generated-artifact records.

## Consequences

- A valid independent document or diagram does not remain invisible merely
  because its first promotion failed.
- Recovery is bounded and deterministic: it is safe to repeat after process
  restart because the generated artifact has stable task/run/key provenance.
- Failures involving core assets, source-derived files, invalid manifests, or
  a second promotion failure remain terminal and require inspection or a new
  learner-authorized task.
- The dispatcher owns one additional recovery branch, covered by a fixture
  that includes an already-completed failed commit journal.

## Rejected Alternatives

- Re-run the CLI automatically.
- Retry all `commit-failed` tasks regardless of output type.
- Copy files directly from an attempt folder into `assets/generated/` outside
  the validator and provenance writer.
- Leave valid generated artifacts discoverable only through raw task folders.
