# Iteration 13 Acceptance Criteria

Status: implementation in progress; criteria are not yet fully satisfied
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: black-box completion conditions for iteration 13.

## AC-13-01 — One conversation creates one learning unit

Given a system-learning project in the new format, when the learner opens it,
then `教师` is the default destination and shows one continuous learner-teacher
conversation with one composer.

The screen does not expose five role selectors, five learning-zone steps, a
task dashboard, a notification center, or a duplicate terminal. `资产` and
`资料` are reachable as quiet peer destinations.

## AC-13-02 — Stream and recover a teacher turn

Given a configured teacher provider, when the learner sends a message, then the
learner message is durably visible before provider completion and the teacher
response streams into one message.

When the learner scrolls away, the viewport is not forced to the bottom. When
the learner stops generation or the stream fails, the partial response has an
honest interrupted or failed boundary. After refresh, durable messages and
their order recover without duplicated text.

## AC-13-03 — Render rich content safely

Given a completed teacher message containing Markdown, code, LaTeX, Mermaid,
SVG, PNG, or JPEG references, then supported content renders in place and its
source remains inspectable.

Incomplete fenced or delimited content stays readable while streaming. Invalid
content falls back to source. Raw HTML, SVG scripts and event handlers,
`foreignObject`, external SVG resources, unsafe URLs, Mermaid script behavior,
and implicit remote tracking-image fetches do not execute.

## AC-13-04 — Show only truthful reasoning information

Given a provider that returns a reasoning summary, then the teacher message may
show it behind a compact expandable `思考` disclosure. Given a provider that
does not return one, no fabricated summary appears. Raw hidden chain of thought
is absent from the API response, local history, logs, and frontend.

## AC-13-05 — Require conversation-native delegation approval

Given a teacher proposal for assistant work, when no later learner message
approves that exact proposal, then no task is created and no CLI is launched.

When the learner revises the proposal, the teacher must disclose the revised
objective and wait again. When the learner approves, the resulting task records
the proposal message, approval message, tool call, task type, objective, and
optional source references.

## AC-13-06 — Keep teacher and assistant responsibilities separate

The teacher can teach and request exactly one assistant handoff in a model
response. It cannot receive shell, file-write, code-execution, task-cancel, or
general-purpose computer tools. A successful handoff says accepted or queued,
not completed.

## AC-13-07 — Enforce simple task admission and concurrency

Given an active task of one primary type in a learning unit, a second task of
the same type is rejected with the existing task identity and state. A terminal
task never blocks a new explicit operation and there is no time cooldown.

No more than two assistant tasks run concurrently for one unit and no more than
five run globally. Different types may coexist within those limits. A repeated
origin operation ID returns the existing task or run rather than creating a
duplicate.

## AC-13-08 — Run assistant work visibly and asynchronously

After durable acceptance and admission, the native CLI opens visibly and the
teacher conversation remains usable. The frontend shows a compact inline
queued/running/terminal status but does not reproduce terminal stdout.

The task can end only as `succeeded`, `partial`, `failed`, or `cancelled` after
running. The only cancellation path is a learner interrupt in the visible
terminal. No frontend or HTTP cancellation control exists.

## AC-13-09 — Report failure and suggest a manual next step

Given an assistant failure, the inline status shows the failed phase, a safe
reason, and a human-readable suggestion for what the learner may publish in a
visible CLI. No automatic retry starts, no retry button or retry API appears,
and the failed task never returns to running. Work that must update LLL is a
new teacher-disclosed and learner-approved task.

## AC-13-10 — Recover task truth after restart

After restart, queued tasks remain eligible; a still-live observed CLI remains
running; a recorded executor that no longer exists becomes failed; and
deterministic validation or commit recovery may finish without invoking the
Agent again. Terminal tasks do not change state during reconciliation.

## AC-13-11 — Accept extensible deliverables only through the workspace contract

An assistant may return files and nested directories for research, code,
experiments, data, images, diagrams, or documents. Only declared, present,
relative, policy-compliant files are promoted. Absolute paths, traversal,
undeclared output, executable SVG behavior, and references outside the attempt
workspace are rejected. Files not declared through the attempt workspace are
never promoted; the handoff contract does not claim OS-level write isolation.

Valid generic deliverables receive stable artifact identities under the
generated-asset area without flattening their internal tree.

## AC-13-12 — Commit editable core assets safely

The active core assets are exactly `引入`, `正文`, and `练习`. A task may report
each as updated, unchanged, or failed and may succeed without updating any core
asset when it delivers other useful artifacts.

Updated or unchanged advances only that asset's conversation cursor. Failed or
conflicted does not. Non-overlapping learner and assistant changes merge;
overlap preserves the learner's current content and reports a conflict. Every
committed change is recoverable as an immutable version with provenance.

There is no draft approval, reject, regenerate, or teacher-review workflow.

## AC-13-13 — Preserve annotation Ask AI

The learner can select body content, create an annotation, ask a question, and
receive a streamed contextual answer. The thread remains associated with the
quote snapshot and asset version. It does not appear in the main conversation,
use the five-role teacher prompt, or invoke an assistant tool.

## AC-13-14 — Store, parse, and protect source material

After explicit confirmation, an accepted source upload is hashed and stored
locally before parsing starts. Common document, spreadsheet, image, text, and
source-code formats create a visible parsing task. Unsupported regular files
remain opaque; archives are not auto-extracted and uploaded programs are not
executed.

Parsing failure preserves the original. Source content is absent from teacher
context until an approved task references it. Cloud processing requires a
learner-visible disclosure before task creation.

The default limit is 100 MiB per uploaded file and 1 GiB of retained source
bytes per learning unit; deployments may lower these values but do not silently
raise them after a request is accepted.

## AC-13-15 — Distinguish source deletion modes

Normal deletion hides and tombstones a source while retaining revisions and
provenance. Explicit permanent deletion removes original and derived bytes and
retains only non-content identity, hashes, timestamps, and a deleted marker.
Generated artifacts that used the source remain assets with provenance.

## AC-13-16 — Compact context without replacing history

When the provider-context estimate reaches 256K tokens or its smaller hard
limit requires earlier action, a learning-aware compact projection is created
outside the assistant queue. Approximately the most recent 64K tokens remain
exact, subject to the provider guard.

The projection records covered event range and hashes, preserves typed learning
state and evidence references, and can be rebuilt. Deleting or corrupting the
projection never deletes canonical conversation events.

## AC-13-17 — Migrate an old system-learning project reversibly

Before changing an old project, migration creates a versioned backup and a
journal. Intro maps to `引入`, Explain maps to `正文`, Practice maps to `练习`,
and Explain confusion/Ask-AI data maps to body annotations.

Summary and Extend are neither copied into `正文` nor supplied to new teacher or
assistant context. Their bytes remain available to rollback. A failed migration
restores the old readable project and does not expose a half-migrated unit.

## AC-13-18 — Preserve discipline-map behavior as soft guidance

Existing discipline overview, topic catalog, learning plan, folder binding,
and learning-scope provenance remain readable. A map may create multiple
learning units, and each unit owns one teacher conversation. Topic and chapter
metadata influence initial teacher context but do not mechanically prevent the
learner from asking across topics.

## AC-13-19 — Keep local files authoritative

Conversation, tasks, runs, sources, assets, versions, and migration state can
be rebuilt from the documented project paths without a database or external
broker. In-memory indexes and rendered projections may be deleted and rebuilt
without losing canonical user content or task truth.

## AC-13-20 — Cut over as one experience

The new frontend becomes default only after migration compatibility, REST
recovery, global SSE invalidation, provider streaming, task recovery, asset
commit recovery, source privacy, and Ask-AI tests pass together.

After cutover, the knowledge-greenhouse and five-zone navigation are hidden.
Rollback can reopen the preserved legacy reader without reversing committed
new conversation or asset files.
