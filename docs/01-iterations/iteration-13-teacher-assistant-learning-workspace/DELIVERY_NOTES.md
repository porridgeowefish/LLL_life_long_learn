# Iteration 13 Delivery Notes

Status: implementation complete; credential-dependent provider and visible-native-CLI smoke remain release checks
Owner: project maintainer
Last reviewed: 2026-08-31
Source of truth: short-lived alignment progress, open decisions, and eventual implementation evidence for iteration 13.

## Alignment Progress

| Contract area | Status | Current focus |
|---|---|---|
| 1. Local conversation storage | aligned | path, events, IDs, recovery, migration, annotations, and compaction confirmed |
| 2. Teacher tool | aligned | minimal args, provider normalization, result semantics, and call limit confirmed |
| 3. Assistant task | aligned | proposals, queueing, concurrency, input sealing, failure guidance, cancellation, idempotency, and recovery confirmed |
| 4. CLI result | aligned | generic deliverables, strict handoff paths, manifest, and optional asset results confirmed |
| 5. Asset commit | aligned | cumulative candidates, deterministic merge, cursors, versions, and recovery confirmed |
| 6. Notification | aligned | conversation cards, lightweight SSE, durable REST, and no separate receipt confirmed |
| 7. Source material | aligned | versioned originals, explicit parsing, privacy, failure, and deletion confirmed |
| 8. Frontend migration | aligned | single chat, rich streaming, compatibility migration, cutover gate, and rollback reader confirmed |
| Iteration document set | complete | required plan, stories, acceptance, interface, data, test, ADR, and long-lived sync created |

## Confirmed Alignment

- The product is reorganized around `教师 / 资产 / 资料` rather than presenting
  five learning zones as the primary experience.
- `教师` is one restrained, OpenAI-style continuous conversation: centered
  readable content, generous whitespace, minimal navigation, and one bottom
  composer. Teacher proposals and learner feedback are ordinary messages.
- Assistant progress is a compact inline system status in that same stream,
  not a visible message category, separate task-card module, side panel,
  dashboard, or notification center. The five teaching roles have no visible
  role-switching UI.
- Teacher output streams into one ordinary message and supports Markdown,
  code fences, LaTeX, Mermaid, sanitized isolated SVG, and referenced PNG/JPEG
  previews. Incomplete rich blocks stay as source placeholders and rendering
  failure never destroys their source.
- An expandable `思考` area contains only provider-supplied reasoning summaries
  and coarse progress states. Raw hidden chain-of-thought is not requested,
  persisted, or exposed, and unsupported providers do not receive a fabricated
  substitute.
- LLL normalizes provider streams into stable content-block events. Canonical
  history keeps final source and attachment references, while rendered HTML,
  Mermaid output, decoded images, and transient progress are projections.
- Rich content is not trusted: raw Markdown HTML is disabled; Mermaid is
  strict; SVG removes executable, external, and navigation behavior and renders
  in isolation; remote image tracking is not fetched implicitly.
- The knowledge-greenhouse frontend presentation is hidden at cutover but is
  not deleted as part of that presentation change.
- The API teacher is the only participant that speaks in the main learning
  conversation.
- CLI assistants work asynchronously and return task results rather than
  injecting messages into the teacher conversation.
- Existing non-greenhouse templates are reusable assistant output targets when
  a scoped task explicitly allows the target.
- Shared template definitions remain read-only; assistants write project-owned
  instances created from those templates.
- Assistant work must be grounded in an explicit teacher task and a bounded
  snapshot of accumulated teacher-learner conversation context. The assistant
  does not independently choose a new learning agenda.
- The teacher's product surface remains teaching-only: it conducts the
  conversation, assesses understanding, adapts instruction, and expresses
  bounded supporting-work needs. It does not operate the CLI, manage files or
  templates, or directly create or modify formal teaching assets.
- LLL, rather than the teacher runtime, converts supporting-work needs into
  authorized durable tasks. CLI assistants asynchronously perform the
  production work and generate asset candidates; Go validates and commits all
  formal asset writes.
- Assistant execution never blocks the teacher's current API response. Results
  are surfaced to the teacher only at a subsequent turn boundary, while task
  progress remains inline and cancellation is only a manual terminal interrupt.
- The earlier draft-first accept/reject/promotion proposal is superseded. A
  successful task commits a validated editable asset version directly. The
  learner edits generated content normally; the teacher does not review or
  manage it, and the UI exposes no reject or regenerate workflow.
- Assistant output has durable task results and editable asset updates. A later
  teacher turn may receive a bounded result summary as context, but there is no
  separate teaching-receipt product.
- The API teacher has no general-purpose execution or file-operation tools and
  must not claim to have run or externally verified something. Its only
  assistant-facing operation is a narrow handoff tool used after an explicit
  learner request to generate material. Small code-run, shell, and interactive
  debugging needs stay in the learner's normal IDE; CLI assistants are reserved
  for substantial asynchronous production work.
- A CLI assistant may execute code and use specialist tools inside an
  authorized substantial deliverable. This does not expose a remote IDE or
  admit small interactive debugging tasks into the assistant queue.
- The complete conversation remains the canonical learning record. Generic
  rolling summaries and per-turn summary deltas cannot replace it. Lossless
  indexes are allowed; any lossy context representation requires a dedicated
  learning-aware compaction design with source provenance and preservation
  criteria.
- The prior no-compaction decision is superseded. Full events remain canonical,
  while a small rebuildable auto-compacted projection may be combined with
  recent exact events and the unified system prompt for teacher context-window
  management.
- The teacher may use one narrow handoff tool when its soft methodology finds
  substantial verification, consolidation, reflection, or material-production
  work suitable for CLI delegation. Learner requests use the same path, while
  LLL does not infer tasks from turn counts, idle timers, or a state machine.
- Complete conversation history is stored locally as project-owned canonical
  data under a stable path contract. Assistant tasks receive explicit
  conversation identity and source scope rather than rediscovering or
  summarizing the UI transcript.
- Learners do not select a summary range. LLL computes an incremental range per
  target asset. The assistant reads the current saved asset plus the exact
  unincorporated conversation turns and commits a cumulative next version. A
  successful commit advances the asset-specific cursor.
- Consolidation is requested in ordinary natural language; the teacher
  composer has no dedicated `沉淀` action. Conversation
  shows a compact non-speaking inline task status; the asset workspace only
  displays and edits committed results.
- The five organizing roles are `引导人 / 澄清者 / 验证者 / 沉淀者 / 复盘者`.
  They must shape both the teacher's pedagogical methods and assistant
  generation; generic fixed asset categories must not silently recreate the
  old five-zone product model.
- The teacher keeps all five roles as a dynamic method library; learners do not
  switch role modes. Roles also inform assistant organization but do not map
  one-to-one to asset columns.
- The five roles are maintained in one versioned teacher system prompt, not as
  separately managed method packages, plugins, prompts, or modes. This unified
  methodology is a soft adaptive default, not an enforced workflow; there is
  no persisted override state or role-entry/exit UI.
- Substantial work arising from verification, consolidation, or reflection may
  be delegated through the teacher's narrow handoff tool. The API model submits
  intent; LLL owns the durable task and CLI execution.
- The teacher handoff targets an application-level assistant-task service, not
  a CLI. CLI runtimes are replaceable executor adapters; the service injects
  physical paths, output schemas, notification anchors, and lifecycle policy.
  Product tasks are durable and distinct from linked CLI execution sessions.
- Executors write task-scoped output manifests. LLL validates and commits asset
  updates, returns immediate task acceptance in the teacher stream, and uses
  persisted query state plus SSE for conversation-anchored inline status.
- A learning unit's single teacher conversation is first-class project data
  under `projects/<unit-slug>/conversation/`; there is no plural conversation
  collection inside a unit. Conversation, assistant
  task, and executor run records remain separate and link by stable IDs.
- Each conversation uses `conversation.json` metadata plus an append-only
  `events.jsonl` timeline containing full messages, response boundaries, tool
  calls, and task links. Task cards are projections joined from durable task
  state rather than copied into conversation history.
- A discipline-backed folder may contain multiple conversation-unit pairs. A
  teacher conversation comes first and forms exactly one learning unit; that
  unit owns the assets, sources, tasks, and progress accumulated through the
  conversation. It never contains multiple teacher conversations.
- Discipline overview topics and existing scope metadata are guidance rather
  than hard teaching boundaries. The teacher may cross related topics without
  being refused or forced into another conversation.
- Existing system-learning projects keep their identity, discipline-folder
  membership, progress, and run history, but their persisted content is
  migrated into the conversation-first learning-unit shape. Existing learning
  content does not need to be regenerated.
- All units converge on one canonical iteration-13 storage shape. Legacy and
  new paths are not retained as permanent parallel writable representations;
  migration must be versioned, restart-safe, auditable, and recoverable.
- Active teaching assets share the extensible `assets/` container and use
  `assets/intro/`, `assets/body/`, and `assets/practice/`. `body` is an ordinary
  editable正文 asset, not a destination for legacy Summary or Extend output;
  new asset types may be added as sibling directories.
- Learning units keep the flat canonical root `projects/<unit-slug>/`.
  Discipline folders remain index-based navigation metadata rather than
  physical parent directories, so reclassification does not move a unit or
  change its stable identity. Iteration 13 normalizes the contents of each root
  rather than introducing a nested discipline hierarchy.
- `conversation/` contains versioned `conversation.json`, append-only
  `events.jsonl`, and rebuildable `compact.json`. JSONL is the machine source of
  truth, Markdown is export-only, provider token deltas are not durable, and a
  single Go repository writer owns sequencing and truncated-tail recovery.
- Conversation history uses six initial semantic events: complete messages,
  teacher-response start and finish boundaries, normalized tool request and
  result, and a stable task link. LLL owns type-prefixed ULIDs and monotonic
  sequence numbers; provider identifiers are external references only.
- Inline selection, annotation, and Ask AI on final teaching assets remains a
  supported capability. Its contextual threads belong to the owning asset and
  do not become extra primary teacher conversations or get merged into the
  main `conversation/events.jsonl`. Legacy `explain/confusions.json` content is
  migrated into canonical body-asset annotation storage with provenance.
- Main teaching and inline annotation Q&A are separate application services.
  Only the teacher loads the five-method prompt, complete primary conversation,
  and assistant-handoff tool. Ask AI uses a small asset-local prompt; both
  reuse the provider gateway, streaming, error handling, credentials, and usage
  accounting infrastructure.
- `assets/body/` is the new正文 and final-note area. Migration converts legacy
  `intro`, `explain`, and structured `practice` content into the three active
  assets; legacy Ask AI becomes body-owned annotation threads. Existing
  standalone `summary` and `extend` output is not merged, displayed, used as
  context, or targeted by new CLI work, and physical cleanup is out of scope.
- Conversation reads use paged REST with `afterSeq` for refresh, reconnect, and
  sequence-gap recovery. SSE only accelerates delivery of committed events;
  the frontend cannot append arbitrary events, task status is joined from the
  task API, and compacted context is never substituted for visible history.
- Learning-aware auto-compaction is asynchronous internal maintenance owned by
  `ConversationCompactionService`. It uses an ordinary model API without file
  tools; Go validates the typed result and atomically writes `compact.json`.
  It does not block the triggering teacher response, invoke CLI, create a task
  card, mutate raw events, or compact asset Ask AI threads.
- Main-dialogue auto-compaction normally starts at an absolute `256K` estimated
  tokens and retains roughly the latest `64K` as exact events. The configured
  provider's real safe input boundary remains a hard guard and may trigger
  earlier compaction for smaller-context models; silent truncation is forbidden.
- `compact.json` preserves typed, evidence-linked learning state rather than a
  prose-only summary. Goals, understanding, misconceptions, open questions,
  verification status, explanations, preferences, commitments, references,
  and critical exact excerpts remain distinguishable and source-traceable.
- The teacher-facing `delegate_learning_work` tool accepts only `objective` and
  optional stable `sourceRefs`, plus the primary operational `taskType` added
  by D13-055 solely for active exclusion and presentation. It has no hard pedagogical
  workflow, paths, ranges, schemas, executor controls, or notification fields;
  LLL injects those details.
- Provider adapters buffer native tool arguments and expose only complete,
  provider-neutral `text-delta`, `tool-call-ready`, `usage`, completion, and
  failure events. TeacherService validates complete JSON before dispatch;
  provider frames and call IDs never become LLL business contracts, and no
  Agent SDK is required.
- Teacher tool results distinguish durable `accepted`, business `rejected`, and
  infrastructure `failed`; accepted never means completed. At most one task is
  accepted per teacher response, while a single objective may combine several
  pedagogical intentions. Retries rely on task-service idempotency.
- Accepted assistant work enters a Go-owned local durable queue; no external
  broker is required. Product tasks survive restart and remain separate from
  one or more replaceable executor attempts. Dispatcher leases and heartbeats
  prevent duplicate execution and recover orphaned attempts.
- Assistant execution defaults to five global slots and two slots per learning
  unit. There is no resource-lock compatibility matrix. Same-type work for one
  unit is excluded only while queued or running; conflicting calls return the
  existing task result, and terminal tasks impose no time cooldown.
- Assistant inputs use two-phase sealing: acceptance fixes intent, conversation
  cutoff, and authorized logical scope; first execution fixes current asset
  base versions and the concrete source revision/hash manifest. Authorized
  files added while queued may enter, and files added after sealing may not.
- Assistant persistence separates mutable product `task.json`, immutable sealed
  `input-manifest.json`, and one executor `attempts/<run-id>/envelope.json` per
  attempt. Executors never update task state or asset cursors; failed and
  cancelled tasks remain terminal.
- The initial product lifecycle is intentionally small: queued, running,
  succeeded, partial, failed, and cancelled, with preparing, executing,
  validating, and committing exposed only as a running phase. CLI tasks require
  explicit tool use or learner action; there is no automatic retry, hidden task
  creation, dependency graph, or task-completion chaining.
- Failure appears as a system notice below the anchored conversation message,
  including a human-readable CLI next-step suggestion. There is no retry
  button, helper command, or retry endpoint. Manual CLI work is not silently
  imported; work returning to LLL becomes a new explicit task.
- Teacher tasks use one primary active-exclusion type: `consolidate`, `verify`, or
  `produce-material`; upload parsing is internally `source-processing`.
  Reflection remains a soft method, mixed tasks choose their primary outcome,
  and type does not create permissions, resource locks, or workflow branches.
- Cancellation exists only as a learner's manual interrupt in the visibly
  launched CLI terminal. There is no frontend cancel control, cancellation API,
  teacher cancel tool, dispatcher kill, or automatic timeout cancellation;
  intentional interruption is cancelled, unexpected process loss is failed.
- Before tool use, the teacher discloses the exact logical assistant
  instruction, expected inspection or output, and its learning value. The same
  objective enters the tool call; no hidden shell command, physical path, or
  guaranteed asset update is presented.
- Every teacher delegation waits for learner feedback after disclosure, even
  when the learner initiated the request or clicked the fixed consolidation
  action. Feedback may approve, revise, or decline; revisions require a new
  disclosed version, and no task, slot, or CLI launch exists before approval.
- The durable teacher proposal message and subsequent learner feedback message
  are the authorization record; there is no proposal tool, file, modal, button,
  or separate state machine. LLL derives both adjacent message IDs from durable
  history, and TaskRecord preserves them with the call ID without exposing an
  internal ID argument to the model.
- Explicit origin operation IDs provide deterministic Task and Run idempotency;
  objective similarity does not. Startup resumes queued authorization, observes
  a still-live executor, fails a missing executor without auto-retry, and may
  resume only deterministic validation or commit recovery without invoking the
  Agent again.
- CLI deliverables are extensible beyond intro, body, and practice, but their
  location is not free-form. Each attempt has documented `scratch/`, declared
  `deliverables/<key>/{artifact.json,files/}`, optional `asset-updates/`, and a
  `result-manifest.json`. Only validated declared outputs are promoted; generic
  artifacts enter `assets/generated/<artifact-id>/` with their tree preserved.
- `result-manifest.json` is a generic delivery receipt with arbitrary declared
  artifacts and optional `updated/unchanged/failed` core asset candidates.
  Updated or unchanged advances that asset's cursor. LLL validates, performs a
  deterministic three-way merge against learner edits, preserves learner
  content on conflict, and creates provenance-rich immutable versions through
  per-asset atomic commits and a recoverable task journal.
- Assistant progress and results appear only as compact system status anchored
  in the conversation. Lightweight SSE invalidates state, REST recovers full
  details, and completion never creates an unsolicited teacher message or a
  separate notification-center product.
- Sources retain local immutable originals and versioned derived parsing under
  a strict path contract. Parsing common documents, images, and source text is
  an explicitly disclosed `source-processing` task; unsupported files remain
  opaque, failure preserves originals, cloud-content use requires disclosure,
  and normal tombstone versus explicit permanent deletion are distinct.
- Obsolete direct-generator JSON fields must not constrain new assistant output.
  Conversation-grounded consolidation may add ordinary conclusions to any
  appropriate active asset, but it does not recreate a standalone Summary
  generator or automatically merge legacy Summary into正文.
- Active cumulative asset destinations are `引入 / 正文 / 练习`. Legacy
  extension and standalone-summary destinations are outside the active surface.
- Legacy `延伸` and standalone `总结` definitions are retained as inactive
  asset templates, hidden from the active frontend and excluded from new
  generation. They are not placed in the learner source-material library and
  are not physically deleted in iteration 13.
- One teacher `tool_use` creates one CLI task. A consolidation task reads the
  new conversation range once, checks `引入 / 正文 / 练习`, and reports an
  `updated / unchanged / failed` outcome per destination rather than launching
  three CLI invocations. Verification and material-production tasks may return
  generic deliverables without core asset updates.
- The new slice includes orchestration extraction, the teacher-to-CLI task
  mechanism, the teaching-asset library, and the source-material library.
- Initial uploaded-file parsing is delegated to an assistant task.

## Frozen Planning Baseline

The decision interview is complete. Product scope and architecture contracts
are frozen in:

- `README.md` for scope, decisions, architecture integration, and waves;
- `USER_STORIES.md` for learner and maintainer value;
- `ACCEPTANCE_CRITERIA.md` for black-box completion;
- `INTERFACE_CONTRACT.md` for HTTP, SSE, provider, tool, task, CLI, asset,
  annotation, and source interfaces;
- `DATA_DESIGN.md` for canonical paths, schemas, migration, and recovery;
- `TEST_PLAN.md` for runnable verification and cutover gates;
- `ADR/0012-teacher-assistant-learning-workspace.md` plus synchronized
  long-lived architecture documents for durable direction.

Implementation may refine internal package names and private algorithms, but it
must not change learner approval, teacher/assistant responsibility, canonical
paths, task lifecycle, source privacy, asset ownership, or compatibility rules
without reopening the affected contract and recording the change here.

## Implementation Decisions Still Left To Code

These are intentionally implementation-local and do not block planning:

- exact Go package splitting sequence inside each delivery wave;
- selected sanitizer, Markdown, KaTeX, Mermaid, and syntax-highlighting
  libraries, provided they satisfy the security and fallback contract;
- exact provider SDK versions and adapter-private diagnostics;
- CSS tokens, typography, and micro-interaction values within the approved
  restrained single-chat direction;
- optional retention cleanup for terminal attempt scratch files, provided the
  first implementation defaults to preservation.

## Implementation Status

Implemented through 2026-08-31:

### Post-implementation interaction refinement

The first refinement pass closes the gap between a functional teacher stream
and a usable daily chat surface:

- an empty learning unit receives one durable, idempotent teacher greeting that
  asks what the learner wants to understand or where they are stuck;
- discipline deep-dive enters the new teacher route directly;
- the native 100% page scale, bounded viewport shell, compact transcript paper,
  and bottom composer keep the input visible on first open;
- the dedicated consolidation button is removed; delegation begins through
  ordinary conversation and retains explicit learner approval;
- file upload and optional asynchronous parsing are available directly from
  the teacher composer;
- provider-supplied reasoning summaries are expandable when present, and no
  substitute is fabricated for providers that do not supply them;
- teacher generation is now owned by a backend response run independent of an
  HTTP subscriber. A page refresh reconnects to `/conversation/responses/active`
  and replays the in-flight response; only the explicit stop action cancels it;
- API model connections have an independent multi-provider page with the
  sequence add, fill/probe, then select in chat. Annotation Ask AI is configured
  as another service binding on that same model page rather than occupying a
  separate navigation entry; search-engine selection remains in unified settings;
- provider selection may be overridden per teacher turn without coupling the
  teacher UI to provider-specific request formats;
- internal message IDs are removed from model-visible context, and delegation
  authorization IDs are derived from adjacent durable history;
- Markdown tables receive explicit borders, spacing, header treatment, and
  alternating rows in teacher, asset, and discipline views;
- legacy structured practice JSON is rendered as exercises, options, answers,
  and explanations rather than displayed as raw source;
- tool-only teacher messages normalize `blocks: null` to an empty list, fixing
  the assistant-invocation white screen and preserving inline task status.

### Explicit assistant-delegation availability fix

A real conversation showed repeated `delegation_not_approved` results before
any task reached the queue. The failure combined three compatibility cases:
the learner approved with an `ok, ...` reply that the narrow phrase matcher did
not recognize; an old teacher message contained a leaked, incorrect
`[messageId=...]` prefix that the provider echoed through the retired tool
argument; and a public paper title was sometimes emitted as a local
`sourceRef`.

The teacher service now recognizes bounded short confirmations such as `ok`,
derives proposal identity only from durable adjacent messages, strips only the
legacy leading message-ID marker from provider context, ignores non-canonical
public reference strings, and continues to strictly validate any canonical
`source_...` local reference against learner selection and disclosure. This
restores explicit delegation without weakening local-file authorization.

A follow-up fix also makes the authorization gate visible and recoverable:
each current turn receives a deterministic allow/deny instruction so an
unrelated message such as “你好” cannot replay stale work; rejected tool calls
persist an explanatory teacher notice instead of disappearing after stream
refresh; the learner can then receive a fresh proposal and approve it normally.
The teacher workspace is widened to an 820 px reading measure and gains a
restrained right rail for message, source, task, current-model, and asset-version
signals. Model selectors display both the friendly name and actual model ID so
an edited model identifier is immediately distinguishable.

The production `分布式计算` transcript exposed a second gate defect: expanded
tool objectives were compared one-way against the shorter approved proposal,
so a more detailed objective lowered its own match ratio and repeatedly caused
`delegation_not_approved`. The approved teacher proposal is now authoritative.
Concise compatible tool wording may be retained; provider expansions are
narrowed back to a complete proposal containing task content, material scope,
deliverables, and learning value. A technical rejection does not revoke the
recorded approval: a subsequent explicit `重试 / 再来 / 继续提交` may reuse that
same unchanged proposal, while an unrelated objective is still rejected.

- a project-owned append-only conversation repository with stable IDs,
  truncated-tail recovery, idempotent learner operations, response replay,
  a learning-aware compact projection at the 256K threshold, and a smaller
  provider-window safety guard that compacts before an oversized request;
- one provider-neutral teacher gateway for OpenAI-compatible and Anthropic
  streaming text, reasoning summaries, and buffered tool calls;
- a focused TeacherService with the unified five-method soft prompt and the
  single `delegate_learning_work` authorization path;
- durable assistant tasks with operation idempotency, same-type exclusion,
  global concurrency five, per-unit concurrency two, restart reconciliation,
  sealed input snapshots, visible native CLI launch, and no automatic retry;
- task attempt workspaces, envelopes, generic deliverables, result-manifest
  validation, generated-artifact promotion, source-derived output promotion,
  and conservative learner-wins asset commits;
- versioned editable `intro / body / practice` assets, source-original
  retention, derived revisions, quotas, tombstone/permanent deletion, and
  active-task deletion protection;
- startup migration with legacy inventory hashes, local backup, journal,
  resumable idempotent canonical initialization, health reporting, and a
  per-project legacy compatibility entry when migration fails;
- the new system-learning frontend organized as `教师 / 资产 / 资料`, with an
  OpenAI-style continuous stream, inline task state, direct asset editing,
  source upload, per-message source selection in the teacher composer, hidden
  greenhouse navigation, and retained legacy body annotation/Ask-AI compatibility;
- Markdown, owned LaTeX preprocessing, strict Mermaid, sanitized fenced SVG,
  and PNG/JPEG/generated-artifact viewing without exposing raw chain of thought;
- service-oriented AI bindings for teacher, annotation Ask AI, and conversation
  compaction while preserving existing Ask-AI settings compatibility.

The implementation reused the existing route-owned backend incrementally:
new route files are thin adapters over repositories and services, while legacy
routes remain available for discipline maps and rollback compatibility. This
keeps the change deployable without rewriting unrelated project, progress, or
session behavior.

## Verification Evidence

Completed on 2026-08-31 from the repository root:

- `go test ./...`: pass;
- `npm --prefix frontend test -- --run`: pass, 40 files and 135 tests;
- `npm --prefix frontend run build`: pass;
- root production build (`npm run build`, including `go build`): pass;
- isolated Playwright browser acceptance (`scripts/qa_iteration13.py`): pass for
  unit creation, body editing, annotation, Ask AI opening, source upload, and
  per-message source selection in the teacher composer; the refinement pass
  additionally verifies the durable greeting, first-viewport composer,
  removed consolidation action, chat upload affordance, unified teacher/Ask AI
  model configuration, and search-engine retention;
- `git diff --check`: pass after the final changes.

New automated coverage includes conversation tail recovery and response
idempotency, teacher streaming persistence and approved delegation, task
idempotency and same-type exclusion, restart/PID identity recovery, asset edit
and interrupted-commit recovery, source quotas/classification/deletion
evidence, migration backup/resume/cutover, server REST/SSE behavior, Ask-AI
partial-failure persistence and hidden-thinking suppression, AI binding
resolution, and active-content removal from fenced SVG.

Still manual because it depends on this machine's configured credentials and
desktop process behavior:

- one real configured provider stream for each enabled provider kind;
- one selected native CLI task from visible terminal launch through a valid
  result manifest and promoted artifact;
- manual Ctrl+C classification in the visible terminal;
- visual QA at narrow viewport sizes.

These are release smoke checks, not missing code paths. They are deliberately
not reported as automated passes.

### Visible assistant CLI correction

The first assistant-task launcher used Claude print mode (`-p`) and Codex
`exec`. Although the process ran in a visible terminal, this reduced the
terminal to a non-interactive background-task shell. The launcher now restores
the product's explicit CLI contract: it opens directly in the isolated task
workspace, reads the durable `prompt.md`, injects that prompt as the initial
interactive turn, and leaves the real Agent CLI visible. The learner can
inspect the session and stop it with Ctrl+C; executor/exit markers and result
manifest validation remain unchanged.

The correction was subsequently promoted from a launcher patch into the
`agentexecution.Service` domain service. Encyclopedia generation, learning
agents, teacher delegation, interactive resume, and existing headless Agent
jobs now enter through that service. Server routes and the assistant dispatcher
no longer own CLI process-launch commands. Visible project and assistant-task
executions converge on the same prompt-injected terminal implementation.

## Planning Validation

Completed on 2026-08-30:

- required iteration files: pass;
- required front matter on all new and synchronized documents: pass;
- relative Markdown links: pass;
- balanced fenced blocks and trailing-whitespace scan: pass;
- stable vocabulary audit for task states, phases, types, concurrency, assets,
  source limits, compaction limits, cancellation, and failure guidance: pass;
- structured reader-question audit: pass after adding the source-derived
  update contract, workspace-wide migration preflight, and terminal failure
  guidance boundary.

Three independent fresh-reader attempts were started under the documentation
coauthoring workflow but could not run because the available agent account hit
its usage limit. This is recorded as an unexecuted independent review, not a
pass. The primary agent completed the same question, consistency, and
implementability checklist locally. A future implementation review should
repeat the independent-reader check when capacity is available.

The original planning-only note is superseded by the implementation and
verification evidence above. Independent-reader review and native desktop
smoke tests remain explicitly separate from the automated implementation pass.
