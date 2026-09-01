# Iteration 13 User Stories

Status: implemented; manual experience validation pending
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: learner-visible value and behavior for iteration 13.

## Primary Journey

### US-13-01 — Learn through one teacher conversation

As a learner, I want one responsive teacher conversation to be the center of a
learning unit, so that teaching feels continuous instead of being divided into
five generated pages and disconnected AI actions.

Value:

- the teacher can explain, ask, clarify, verify understanding, and adapt;
- one conversation forms one learning unit;
- discipline and chapter metadata guide the conversation without forbidding a
  natural change of topic.

### US-13-02 — Read rich streamed teaching content

As a learner, I want teacher responses to stream and render Markdown, code,
LaTeX, Mermaid, SVG, PNG, and JPEG safely, so that explanations remain fast and
can use the notation and media a subject requires.

I can inspect a provider-supplied reasoning summary when one exists, stop a
streaming response, scroll independently, and recover the finished conversation
after refresh. I am never shown a fabricated or raw hidden chain of thought.

### US-13-03 — Know before heavy assistant work begins

As a learner, I want the teacher to explain the proposed CLI work, expected
output, inputs, and learning value before delegating, so that an asynchronous
task is never launched as a hidden side effect.

My ordinary next message can approve, revise, or decline. Revision requires a
new proposal; decline creates no task.

### US-13-04 — Continue learning while an assistant works

As a learner, I want substantial verification, consolidation, research,
experiments, diagrams, and material production to run asynchronously in a
visible CLI, so that the teacher remains responsive while the heavy work is
inspectable.

The conversation contains only a compact inline status. It does not become a
task dashboard or a duplicate terminal.

### US-13-05 — Understand failure without automatic behavior

As a learner, I want a failed assistant task to report its stage, safe reason,
and a suggested manual CLI next-step instruction, so that I can decide what to
do without the system silently spending more time or changing files.

I cancel only by manually interrupting the visibly running terminal. There is
no misleading frontend cancel or retry button.

## Durable Learning

### US-13-06 — Keep the complete learning record locally

As a learner, I want complete teacher and learner messages, approvals, tool
calls, and task links stored inside the learning unit, so that I can inspect,
back up, and recover the learning history without a cloud account.

Long conversations may use a learning-aware compact projection for model
context, but the original events remain canonical and auditable.

### US-13-07 — Accumulate editable teaching assets

As a learner, I want the conversation to accumulate editable `引入`, `正文`, and
`练习` assets, so that useful teaching becomes durable material rather than a
transient transcript.

Asset updates are cumulative and versioned. They may be unchanged when the new
conversation adds no relevant content. My current edits win when an assistant
candidate overlaps them.

### US-13-08 — Keep arbitrary useful assistant deliverables

As a learner, I want an assistant to return research, source trees, code,
experiment output, datasets, images, diagrams, and documents without forcing
everything into three note files, so that the assistant remains capable of
real supporting work.

Every deliverable still lands through a documented staging, declaration,
validation, and promotion contract.

### US-13-09 — Ask questions inside the final note

As a learner, I want to select or annotate content in the body asset and use
Ask AI there, so that a local question does not require restarting the main
teacher flow.

The annotation conversation uses the selected quote, nearby asset content,
asset version, and its own thread. It does not load teacher methodology or
assistant tools.

## Source Material

### US-13-10 — Add private source material explicitly

As a learner, I want to upload a document or file, see what parsing will do,
and approve any cloud disclosure, so that source processing is useful without
quietly leaking content or treating an upload as teacher context.

The original is retained locally before parsing. Parsing failure does not
destroy it, unsupported formats remain available as opaque files, and only an
authorized task may read source content.

### US-13-11 — Manage source revisions and deletion

As a learner, I want replacements to create source revisions and normal
deletion to be recoverable, so that provenance remains meaningful. I can also
request explicit permanent byte deletion when that is my intent.

## Compatibility And Navigation

### US-13-12 — Keep discipline orientation without rigid teaching scope

As a learner, I want discipline overview, learning plan, and topic-boundary
metadata to remain available, so that I retain orientation while the teacher
can still follow my real questions.

A discipline map may lead to many learning units. Each unit has one main
teacher conversation and one soft scope, not one conversation per topic node.

### US-13-13 — Open an old project in the new experience

As an existing learner, I want my Intro, Explain, Practice, and Ask-AI content
to appear in the new asset model after a safe migration, so that the redesign
does not discard prior work.

Legacy Summary, Extend, and knowledge-greenhouse data remain on disk for
rollback but are not injected into the active body or teacher context.

### US-13-14 — Edit assets without workflow ceremony

As a learner, I want committed teaching assets to be ordinary editable files,
so that I do not have to accept, reject, regenerate, or curate an intermediate
AI draft before using or correcting it.

## Operator And Maintainer Stories

### US-13-15 — Recover local work after application restart

As the local operator, I want queued work, live CLI observation, failed missing
executors, and deterministic validation or commit recovery to reconcile after
restart, so that the filesystem never lies about task state.

### US-13-16 — Add providers and executors behind stable contracts

As a maintainer, I want provider-native streams and CLI-specific details behind
normalized interfaces, so that TeacherService, persistence, and the frontend do
not depend on one SDK or one Agent runtime.

### US-13-17 — Evolve the Go backend without route-owned orchestration

As a maintainer, I want HTTP handlers to delegate to application services and
repositories, so that teacher turns, task admission, source parsing, and asset
commits are testable without constructing HTTP requests or launching a real
provider.

## Story Boundaries

These stories do not authorize multi-user collaboration, cloud sync, an
embedded terminal, automatic assistant work, automatic retry, workflow graphs,
raw chain-of-thought disclosure, database migration, or physical deletion of
legacy zone data during the iteration-13 cutover.
