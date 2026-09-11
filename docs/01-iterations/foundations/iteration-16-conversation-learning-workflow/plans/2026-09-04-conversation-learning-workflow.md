# Conversation Learning Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete source reference, teaching boundary, consolidation, and teacher-usage workflows without restoring retired zones.

**Architecture:** Reuse the existing file-first source revisions and sealed assistant task model. Add small source, scope, usage, and UI contracts; keep the teacher as the only classroom voice and the assistant as a confirmed asynchronous worker.

**Tech Stack:** Go, React 18, TypeScript, Vitest, file-first JSON/JSONL.

**Spec:** `docs/01-iterations/iteration-16-conversation-learning-workflow/README.md`

## Global Constraints

- Never recreate Summary, Extend, Knowledge Garden, or assistant token UI.
- Only teacher provider-reported tokens are persisted; do not estimate missing usage.
- Source parsing retains exactly `content.md`; original uploads remain immutable.
- Consolidation defaults to Intro + Body and requires explicit question intent for Practice.
- Every code task starts red and ends green.

---

### Task 1: Simplify source parsing and composer citation

**Files:** source store/tests, assistant task validation/commit/tests, source HTTP routes/tests, `SourcesView`, `TeacherView`, learning workspace client/tests.

- [x] Write failing tests for one-file parse output, content endpoint, processing status, and citation chips.
- [x] Run focused tests and observe current multi-derived/preview behavior fail expectations.
- [x] Enforce `content.md`, expose it only to the citation picker, and render Sources as a name/type/status list.
- [x] Run focused Go and Vitest suites.

### Task 2: Persist and inject teaching outline

**Files:** learning-scope catalog/snapshot code/tests, teacher context/tests, project creation route/client types.

- [x] Write failing map-origin scope/context tests for `teachingOutline`.
- [x] Persist selected topic prose into the scope snapshot and append a bounded outline section to teacher system context.
- [x] Run focused scope and teacher tests.

### Task 3: Make consolidation a deliberate Intro/Body task

**Files:** teacher delegation schema/tests, assistant task prompt/validation/commit/tests.

- [x] Write failing tests that distinguish ordinary consolidation from a requested question-producing consolidation.
- [x] Add a narrow consolidation mode to the teacher/assistant contract and Chinese prompt rules: synthesize covered content into Intro significance and Body manuscript plus critical-thinking conclusion.
- [x] Permit Practice updates only in explicit question mode and run task/teacher tests.

### Task 4: Persist teacher usage and replace Agent management with usage

**Files:** teacher conversation usage store/tests, provider/gateway/service/tests, HTTP routes/tests, frontend usage feature/tests, app router/sidebar.

- [x] Write failing tests for usage append/pagination and no `/agents` route.
- [x] Record provider `usage` frames per completed response, add paginated teacher-usage API/UI, and remove Agent page/navigation.
- [x] Run focused Go and Vitest suites.

### Task 5: Documentation and full verification

**Files:** ADR-0017, ADR index, long-lived fact owners, iteration index/docs.

- [x] Synchronize documentation and record automated verification in delivery notes.
- [ ] Run native browser E2E (learner-owned acceptance).
