# Retire Legacy Summary, Extend, and Knowledge Garden Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove inactive Summary, Extend, and Knowledge Garden product code and interfaces without deleting historical project files.

**Architecture:** The active learning workspace does not consume these zones. Delete their isolated frontend feature code, unmounted Greenhouse page, old five-zone compatibility branches, backend flashcard surface, workspace scaffolding, and role registrations. Preserve historical files by making migration ignore—not delete—the retired directories; old URLs fall back to Explain instead of rendering a retired zone.

**Tech Stack:** React 18, TypeScript, Vite/Vitest, Go `net/http`, file-first project storage.

**Spec:** `docs/01-iterations/iteration-15-retire-legacy-summary-extend/README.md`

## Global Constraints

- Preserve `projects/<slug>/summary/**` and `projects/<slug>/extend/**` exactly.
- Do not remove provider reasoning summaries, Ask-AI summaries, task result summaries, or Explain critical-summary pages.
- Do not change active Teacher, Assets, Sources, Intro, Explain, Practice, or encyclopedia behavior.
- Use tests before each production-code change; record the red and green command output in the execution log.
- Keep all planning and delivery documents under this iteration directory.

---

### Task 1: Remove the frontend retired zones and Knowledge Garden

**Files:**
- Delete: `frontend/src/features/legacy-zones/components/summary/**`
- Delete: `frontend/src/features/legacy-zones/components/extend/**`
- Delete: `frontend/src/features/legacy-zones/api/flashcards.ts`
- Delete: `frontend/src/features/legacy-zones/api/extendFlowers.ts`
- Delete: `frontend/src/features/projects/GreenhousePage.tsx`
- Delete: `frontend/src/features/projects/GreenhousePage.module.css`
- Modify: `frontend/src/features/legacy-zones/index.ts`
- Modify: `frontend/src/features/projects/components/LegacyProjectView.tsx`
- Modify: `frontend/src/features/projects/components/ZoneTimeline.tsx`
- Modify: `frontend/src/shared/types/domain.ts`
- Create: `frontend/src/features/legacy-zones/index.test.ts`

**Interfaces:**
- Consumes: `PracticeFlow` remains the only legacy-zone component rendered by the active Assets view.
- Produces: no active import/export for `SummaryPage`, `ExtendPage`,
  `KnowledgeFlowerGlyph`, flashcard hooks, or flower hooks.

- [x] **Step 1: Write the failing frontend absence test**

Create `index.test.ts` with this public-boundary assertion:

```ts
import * as legacyZones from './index';

test('does not expose retired Summary, Extend, or Knowledge Garden APIs', () => {
  expect(legacyZones).not.toHaveProperty('SummaryPage');
  expect(legacyZones).not.toHaveProperty('ExtendPage');
  expect(legacyZones).not.toHaveProperty('KnowledgeFlowerGlyph');
  expect(legacyZones).not.toHaveProperty('useFlashcards');
  expect(legacyZones).not.toHaveProperty('useExtendFlower');
  expect(legacyZones).not.toHaveProperty('useSaveExtendFlower');
});
```

- [x] **Step 2: Run the test to verify it fails**

Run: `npm --prefix frontend test -- --run src/features/legacy-zones/index.test.ts`

Expected: FAIL because the legacy-zone barrel still exports retired symbols.

- [x] **Step 3: Delete retired files and narrow the barrel**

Delete the Summary and Extend component trees, their dedicated hooks, and the
unmounted Greenhouse page. In `index.ts`, delete only the retired exports;
retain `PracticeFlow`, `IntroPage`, confusion, learning-artifact, practice,
file, and progress exports. Narrow `ZoneName`, timelines, and the legacy
fallback to Intro/Explain/Practice; a historical Extend/Summary URL renders
Explain instead of resurrecting its old page.

- [x] **Step 4: Run focused frontend tests**

Run: `npm --prefix frontend test -- --run src/features/legacy-zones/index.test.ts src/features/learning/components/AssetsView.test.tsx`

Expected: PASS.

### Task 2: Remove retired backend APIs and block retired generic-file access

**Files:**
- Delete: `backend-go/internal/compatibility/flashcardstore/store.go`
- Delete: `backend-go/internal/compatibility/flashcardstore/store_test.go`
- Delete: `backend-go/internal/transport/httpserver/routes_summary.go`
- Modify: `backend-go/internal/transport/httpserver/router.go`
- Modify: `backend-go/internal/transport/httpserver/routes_projects.go`
- Modify: `backend-go/internal/transport/httpserver/routes_confusions_test.go`
- Modify: `backend-go/internal/transport/httpserver/contract_freeze_test.go`
- Create: `backend-go/internal/transport/httpserver/routes_retired_zones_test.go`

**Interfaces:**
- Consumes: generic `/files/projects/{id}/...` access list in
  `routes_projects.go`.
- Produces: Summary flashcard endpoints are absent and generic file access
  rejects `summary/**` and `extend/**`.

- [x] **Step 1: Write failing HTTP behavior tests**

Create tests that seed a project with `summary/flashcards.json` and
`extend/flower.json`, then assert:

```go
func assertRouteStatus(t *testing.T, h http.Handler, method, target string, want int) {
    t.Helper()
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
    if rec.Code != want {
        t.Fatalf("%s %s: got %d, want %d", method, target, rec.Code, want)
    }
}

assertRouteStatus(t, server.Handler(), http.MethodGet, "/api/projects/example/summary/flashcards", http.StatusNotFound)
assertRouteStatus(t, server.Handler(), http.MethodGet, "/files/projects/example/summary/flashcards.json", http.StatusForbidden)
assertRouteStatus(t, server.Handler(), http.MethodPost, "/files/projects/example/extend/flower.json", http.StatusForbidden)
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./backend-go/internal/transport/httpserver -run 'TestRetiredZone' -count=1`

Expected: FAIL because registered flashcard routes and generic file access still exist.

- [x] **Step 3: Remove handlers and whitelist entries**

Remove the two router registrations, delete `routes_summary.go` and
`flashcardstore`, remove Summary/Extend from generic read/write allowlists, and
remove obsolete handler tests/contract entries.

- [x] **Step 4: Run focused Go tests**

Run: `go test ./backend-go/internal/transport/httpserver -run 'TestRetiredZone|Test.*File' -count=1`

Expected: PASS.

### Task 3: Retire registered roles and preserve migration history

**Files:**
- Delete: `agents/registry/summary.json`
- Delete: `agents/registry/extend.json`
- Delete: `agents/charters/summary.md`
- Delete: `agents/charters/extend.md`
- Delete: `agents/primitives/review_pack.md`
- Delete: `agents/primitives/critical_thinking.md`
- Modify: `agents/primitives/README.md`
- Modify: `backend-go/internal/compatibility/iteration13migration/migration.go`
- Modify: `backend-go/internal/compatibility/iteration13migration/migration_test.go`
- Modify: `backend-go/internal/modules/projects/internal/workspace/workspace.go`
- Modify: `backend-go/internal/modules/projects/internal/workspace/workspace_test.go`
- Modify: `backend-go/internal/compatibility/artifactwatch/zone.go`
- Modify: `backend-go/internal/modules/assistant/internal/prompt/assembly.go`
- Modify: `backend-go/internal/modules/assistant/internal/prompt/assembly_test.go`

**Interfaces:**
- Consumes: Iteration-13 migration inventories legacy paths but must not delete
  them.
- Produces: Summary/Extend cannot be loaded as registered roles; a migration
  still succeeds with historical directories present.

- [x] **Step 1: Write a failing migration preservation test**

Add a fixture with files under `summary/legacy.md` and `extend/legacy.md`. Run
the migration and assert both paths still exist byte-for-byte, while the
migration inventory has no Summary/Extend entries.

- [x] **Step 2: Run it to verify it fails**

Run: `go test ./backend-go/internal/compatibility/iteration13migration -run TestMigrationIgnoresRetiredZoneFiles -count=1`

Expected: FAIL because `inventoryLegacy` includes both directories.

- [x] **Step 3: Remove roles and narrow migration inventory**

Delete the two role definitions, their charters and dedicated primitives. Remove
their registry/prompt tests. Reduce `ZoneName`, project skeleton creation,
generated-zone detection, predecessor resolution, and artifact watching to
Intro/Explain/Practice. Change the migration inventory to visit only retained
compatibility directories; it must never delete any directory it no longer
inventories.

- [x] **Step 4: Run focused Go tests**

Run: `go test ./backend-go/internal/compatibility/iteration13migration ./backend-go/internal/modules/assistant/internal/prompt ./backend-go/internal/modules/assistant/internal/registry -count=1`

Expected: PASS.

### Task 4: Synchronize current documentation and verification contracts

**Files:**
- Create: `docs/00-product-and-architecture/ADR/0016-retire-legacy-summary-extend-and-knowledge-garden.md`
- Modify: `docs/00-product-and-architecture/ADR/README.md`
- Modify: `docs/00-product-and-architecture/{PRD.md,DOMAIN_MODEL.md,LEARNING_PROJECT_STRUCTURE.md,ROADMAP.md,DATA_MODEL.md,SYSTEM_ARCHITECTURE.md,BACKEND_ARCHITECTURE.md,AGENT_ARCHITECTURE.md,API_CONTRACT_STRATEGY.md,REPOSITORY_MAP.md}`
- Modify: `docs/01-iterations/README.md`
- Modify: `docs/01-iterations/iteration-15-retire-legacy-summary-extend/{README.md,DELIVERY_NOTES.md,TEST_PLAN.md,ACCEPTANCE_CRITERIA.md}`

**Interfaces:**
- Consumes: implementation facts from Tasks 1–3.
- Produces: one current source of truth that names the active product surface,
  retired APIs, and historical-data preservation policy.

- [x] **Step 1: Update ADR and long-lived owners**

Record that retirement removes active code and public interfaces but preserves
historical on-disk files. Remove statements that promise Summary, Extend, or
Knowledge Garden as a compatibility feature. Keep narrowly worded historical
notes only where needed to explain preserved project directories.

- [x] **Step 2: Verify documentation links and repository checks**

Run:

```text
rg -n "0016-retire-legacy-summary-extend-and-knowledge-garden" docs/00-product-and-architecture/ADR/README.md
rg -n "iteration-15-retire-legacy-summary-extend" docs/01-iterations/README.md
npm run check:full
```

Expected: both index searches find their required entry; the repository check
passes. Record the exact command result and any environmental limitation in
delivery notes.
