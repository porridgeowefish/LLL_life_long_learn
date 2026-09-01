# Iteration 13 Test Plan

Status: automated verification complete; native provider/CLI smoke pending
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: verification strategy and runnable completion paths for iteration 13.

## 1. Test Strategy

Iteration 13 changes the primary journey and several persistence boundaries.
Passing isolated unit tests is insufficient. Verification proceeds from pure
schemas and repositories to application services, HTTP/SSE adapters, frontend
rendering, native-CLI integration, migration fixtures, restart recovery, and a
single end-to-end cutover path.

Provider and CLI tests use deterministic fakes by default. A small manual smoke
test uses one configured real provider and the selected native CLI because
stream framing, visible terminal behavior, interruption, and Windows process
observation cannot be proven by mocks alone.

## 2. Required Commands

Run from the repository root:

```powershell
go test ./...
npm --prefix frontend test -- --run
npm --prefix frontend run build
```

If root package scripts wrap these commands, the implementation may add a
single iteration-13 verification script, but it must call the same suites and
return a failing exit code when any step fails.

Before delivery, also run:

```powershell
git diff --check
```

Documentation link validation may use the repository's existing checker or a
small read-only script added with the implementation.

## 3. Go Unit Tests

### Conversation repository

- append all six semantic event types and preserve monotonic sequence;
- concurrent appends serialize without duplicate IDs or sequence values;
- a truncated final JSONL line is ignored and metadata repaired;
- corruption before the final line blocks writes;
- replay reconstructs messages, response status, tool interaction, and task
  links without provider frames;
- repeated operation IDs return the same learner turn;
- rendered HTML and raw hidden reasoning have no persisted field.

### Learning-aware compaction

- no compaction below the configured threshold;
- 256K threshold and smaller provider hard guard both trigger correctly;
- recent exact window remains bounded around 64K tokens;
- projection includes covered range and event hash;
- stale prompt version or hash rebuilds the projection;
- deleting `compact.json` leaves full history readable;
- fixtures preserve unresolved questions, misconceptions, definitions,
  formulas, code, verification state, and their evidence references;
- annotation Ask-AI messages are excluded.

### Provider gateway

- OpenAI- and Anthropic-style fake streams normalize to the same event union;
- fragmented tool arguments emit one `tool-call-ready` only after valid JSON;
- reasoning-summary output is distinct from text;
- unsupported reasoning summaries produce no fabricated event;
- usage and failure translate without exposing native payloads;
- tool result continuation is correctly adapted;
- cancellation produces an interrupted response when supported.

### Delegation and task admission

- valid proposal and later learner approval create one durable task;
- missing, reversed, cross-unit, revised, or declined authorization rejects;
- objective and proposal consistency boundary is enforced by the teacher loop;
- one response accepts at most one task;
- same unit and type rejects while queued or running and returns existing ID;
- terminal same-type work allows an immediate new operation;
- different types admit up to two per unit;
- global execution never exceeds five;
- repeated origin operation ID is idempotent;
- concurrent admission cannot create duplicate same-type tasks.

### Task dispatcher and restart

- queue ordering is stable and rebuildable from task files;
- task becomes running only after lease, input sealing, and attempt creation;
- queued tasks survive restart;
- live process plus heartbeat remains observed;
- missing process becomes `executor-interrupted` failure without auto-retry;
- terminal states remain unchanged;
- validation and commit recovery resume without invoking fake CLI again;
- intentional visible-terminal interruption becomes cancelled;
- unexpected process loss becomes failed;
- no public cancel or retry service exists.
- failed and cancelled tasks remain terminal, and no helper or hidden control
  token can revive them;

### Workspace and manifest validation

- valid nested research, code, data, image, and document trees pass;
- absolute, drive, UNC, traversal, reserved-device, external symlink, socket,
  and undeclared paths fail;
- missing descriptor, entry point, candidate, or manifest fails;
- SVG scripts, event handlers, external resources, and `foreignObject` sanitize
  or reject according to policy;
- deliverable-only success is valid;
- updated, unchanged, and failed asset outcomes validate independently;
- executor writes cannot mutate task, conversation, source, or formal assets.

### Asset repository and commit

- learner PUT with correct revision creates an immutable version;
- stale learner revision returns conflict without overwrite;
- external `current.md` edit imports as a learner version;
- assistant non-overlap merges and creates provenance;
- overlap preserves learner content and does not advance cursor;
- unchanged advances cursor without a content version;
- partial commit keeps successful target and reports failed target;
- crashes at each journal step complete or restore deterministically;
- current content, metadata hash, and version content always agree after
  recovery.

### Source repository

- upload saves and hashes original before task creation;
- repeated operation ID does not duplicate bytes or task;
- per-file and per-unit limits reject before partial acceptance;
- revision replacement preserves older revision;
- unsupported regular file becomes opaque;
- archive is not extracted and executable is not run;
- parsing failure preserves original;
- source content is absent from teacher context without authorization;
- cloud parser requires disclosure acceptance;
- tombstone retains bytes and provenance;
- permanent deletion removes bytes and blocks while a sealed task references
  the revision.

### Migration

- fixtures cover empty, Intro-only, Explain multipage, Practice structured,
  Ask-AI/confusions, fully populated, and malformed legacy projects;
- active content maps only to intro/body/practice and annotations;
- Summary and Extend never enter body, conversation, or source context;
- backup hashes match legacy inputs;
- re-running completed migration is idempotent;
- failure before promotion leaves legacy untouched;
- failure during each promotion step recovers or restores;
- discipline overview, topic catalog, plan, scope, and folder membership remain
  unchanged.

## 4. HTTP And SSE Integration Tests

- conversation read pagination is stable across new appends;
- teacher turn persists learner message before fake-provider completion;
- SSE deltas concatenate by block ID without duplication;
- stream disconnect followed by REST read returns one correct final or
  interrupted message;
- stop-generation affects only the teacher response;
- tool accepted/rejected/failed frames match durable task state;
- task, asset, source, and conversation invalidation events contain IDs only;
- reconnect or missed SSE is repaired by REST;
- new errors use stable codes and do not expose absolute paths or private
  provider payloads;
- legacy confusion routes and new annotation routes resolve the same migrated
  thread;
- no HTTP task-cancel or task-retry route is registered.

## 5. Frontend Unit And Component Tests

### Single teacher surface

- system-learning route defaults to Teacher;
- one message column and one composer render;
- teacher proposals and learner approvals use ordinary message components;
- task status renders inline and does not open another SSE connection;
- no visible five-role controls, five-zone timeline, task dashboard, terminal
  drawer, or notification center appears in the new route;
- Assets and Sources remain reachable and keyboard accessible.

### Streaming behavior

- text deltas update one message without replacing stable prior blocks;
- burst text/reasoning deltas are render-buffered and autoscroll performs at
  most one layout write per animation frame;
- autoscroll follows only when the learner is already near the bottom;
- manual scroll remains respected;
- stop action displays only for an active provider response;
- refresh hydrates from REST without replay duplication;
- interrupted and failed responses preserve partial source.

### Rich renderer

- Markdown headings, tables, links, lists, and code fences render;
- LaTeX backslashes survive Markdown tokenization using owned preprocessing;
- closed Mermaid renders and incomplete Mermaid stays source;
- sanitized SVG renders in isolation;
- invalid Mermaid/SVG/LaTeX falls back to source;
- PNG/JPEG preview lazy-loads, zooms, opens, and downloads;
- remote image URL does not fetch implicitly;
- reasoning disclosure appears only when a summary block exists;
- keyboard navigation, focus, contrast, reduced motion, and screen-reader labels
  pass component checks.

### Assets, annotations, and sources

- learner edit sends base revision and handles conflict without data loss;
- migration-authored multipage body renders page navigation and page content,
  never a `manifest.json` pointer or internal file path;
- migration-authored practice renders interactive questions through the
  tasks/answer-key contract, keeping answer keys private while returning
  objective explanations after submission;
- learner or assistant replacement of a migrated structured asset switches to
  canonical Markdown and is not shadowed by stale legacy files;
- version and provenance views use durable REST data;
- detached annotation retains quote snapshot and Ask-AI thread;
- upload confirmation communicates parsing and cloud disclosure;
- processing, ready, opaque, failed, tombstoned, and deleted states render;
- old Summary, Extend, and greenhouse entries do not appear in active routes.

## 6. Browser Smoke Paths

Run with the frontend development server and Go backend, using the existing
single app-shell SSE connection.

### Path A — Teacher and rich content

1. Open a migrated or new learning unit.
2. Send a question that produces Markdown, math, Mermaid, SVG, and an image
   reference.
3. Observe streaming, scroll away, return, expand reasoning summary if
   provided, and refresh.
4. Confirm one recovered message and safe rendering fallbacks.

### Path B — Approved assistant delegation

1. Ask for a substantial verification or consolidation.
2. Confirm the teacher first discloses objective, input, output, and value.
3. Approve in the next ordinary message.
4. Confirm one queued task, visible native CLI, responsive teacher chat, and
   inline state transition.
5. Complete the CLI task and inspect promoted deliverable and provenance.

### Path C — Failure and manual next step

1. Use a deterministic failing CLI fixture.
2. Confirm failure notice and suggested instruction, with no auto-retry/button.
3. Confirm the suggestion is human-readable and can be copied to a visible CLI.
4. Confirm the failed task remains terminal and no new run is created by LLL.
5. If the work must return to LLL, disclose and approve a new explicit task.

### Path D — Learner edit conflict

1. Start an asset-updating task.
2. Edit the same body passage while the task runs.
3. Finish the assistant candidate with overlapping changes.
4. Confirm learner text remains current, conflict is reported, and cursor does
   not advance for body.

### Path E — Source privacy

1. Upload and explicitly approve parsing of a supported local file.
2. Confirm original storage precedes the parsing task.
3. Verify content is not present in an unrelated teacher request.
4. Approve a source-referencing assistant proposal and confirm the sealed
   revision/hash.
5. Exercise tombstone and permanent deletion behavior.

### Path F — Migration and rollback reader

1. Copy a representative legacy project fixture.
2. Run migration and inspect intro/body/practice/annotations.
3. Confirm Summary and Extend remain only in backup/legacy storage.
4. Simulate cutover rollback and reopen legacy data without removing new files.

## 7. Manual Native-CLI Checks On Windows

- visible PowerShell or configured terminal runs the actual native CLI, not a
  tailed log simulation;
- the user can manually interrupt it and task state becomes cancelled;
- Go-generated PowerShell scripts containing Chinese paths have UTF-8 BOM;
- prompt files are read with explicit UTF-8;
- long prompts containing ASCII double quotes are not passed as one unsafe
  PowerShell 5.1 native argv string;
- stdout and stderr remain separately inspectable after completion;
- frontend never mirrors raw terminal output.

## 8. Security Tests

- Markdown HTML injection fixtures;
- Mermaid link/script and oversized graph fixtures;
- SVG script, event attribute, CSS URL, external image, data URL,
  `foreignObject`, navigation, and entity-expansion fixtures;
- decompression-bomb and archive traversal fixtures remain opaque;
- oversized image dimensions fail bounded decoding;
- source and artifact file-key traversal fails;
- source-processing output can update only its sealed revision and a teacher
  task cannot declare a source update;
- log and error snapshots contain no credentials, auth headers, full private
  uploaded bytes, or hidden provider reasoning;
- permanent deletion cannot target outside one validated source revision.

## 9. Performance And Reliability Budgets

- learner message durable append completes before provider start;
- local first streaming frame target is under 750 ms after the provider emits
  its first delta; provider network latency is measured separately;
- incremental message rendering avoids full-conversation rerender per delta;
- conversation page load reads a bounded projection, not the whole JSONL;
- queue rebuild scans task metadata without reading attempt payload trees;
- asset and source listings read metadata without decoding binary files;
- one lost SSE connection creates no additional EventSource instance.

These are regression budgets, not promises about external model latency.

## 10. Acceptance Traceability

| Acceptance | Primary verification |
|---|---|
| AC-13-01–04 | frontend components, provider integration, Browser Path A |
| AC-13-05–10 | delegation/dispatcher units, HTTP integration, Paths B–C |
| AC-13-11–12 | manifest/asset units, Paths B and D |
| AC-13-13 | annotation service and frontend component tests |
| AC-13-14–15 | source units, HTTP integration, Path E |
| AC-13-16 | compaction fixtures and recovery tests |
| AC-13-17–18 | migration fixtures and Path F |
| AC-13-19–20 | restart recovery and full cutover smoke suite |

## 11. Delivery Evidence

Planning creates no passing implementation evidence. During delivery,
`DELIVERY_NOTES.md` must record command, date, result, environment, skipped
manual paths, and residual risk. The iteration may not be marked implemented
from document checks alone.
