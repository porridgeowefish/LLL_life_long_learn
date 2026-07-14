# Iteration 07 Test Plan

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: planned verification for iteration 07; test code and recorded delivery results own executed truth.

## Backend

- [US-07.1] Canonical project type submitted -> matching project skeleton is created.
- [US-07.1] Unknown project type submitted -> request is rejected with a stable error.
- [US-07.7] Legacy project has no `projectType` -> it decodes as `system-learning`.
- [US-07.3] Discipline map is created -> no learner-facing five-zone artifacts exist.
- [US-07.3] Discipline map is created -> `project.md` uses the map brief and ignores submitted ability, completion, and phase context.
- [US-07.5] Confirmed map-origin request -> one ordinary flat system-learning project is created.
- [US-07.5] Map-origin project is created -> no `subprojects/` directory is written.
- [US-07.4] Overview contains headings -> no folder or project record is created from them.
- Existing project type change is requested -> mutation is rejected.
- Other project progress changes -> map overview bytes remain unchanged.
- Projects in different sidebar folders -> prompt inputs and token policy are identical.
- Indexed discipline map -> folder store creates or promotes one same-name map-backed folder.
- Map-backed folder -> bound map slug never appears in `slugOrder`.
- No explicit map-update request exists -> map regeneration does not run.

## AI choice advisor

Golden cases:

- [US-07.2] `博弈论 + 想知道有哪些领域` -> recommends `discipline-map`.
- [US-07.2] `纳什均衡 + 要会求解` -> recommends `system-learning`.
- [US-07.2] Information is insufficient -> asks one clarification question and omits a forced recommendation.
- [US-07.2] Long-term and current goals conflict -> returns one recommendation plus a clear trade-off.
- [US-07.2] Advice returns -> no project is created and both choices remain available.
- [US-07.2] Dialog closes -> temporary conversation is cleared and never persisted.
- [US-07.2] Evidence is missing -> learner background is not invented.
- [US-07.2] Advice is rendered -> text remains concise and does not become topic teaching.

## Prerequisite bridge

- [US-07.8] Weak or missing prerequisite is selected -> concise bridge content is returned inline.
- [US-07.8] Bridge succeeds -> no project, folder, or sidebar entry is created.
- [US-07.8] Bridge renders -> one lightweight check and a return-to-topic cue are visible.
- [US-07.8] AI is unavailable -> learner remains in the current project and receives a recoverable error.

## Discipline overview contract

Use representative broad fields such as physics, game theory, computer
science, and philosophy. Verify that the overview:

- Broad discipline is generated -> overview establishes purpose and boundaries.
- Broad discipline is generated -> major areas are covered without deep tutorials for each.
- Overview is generated -> at least one meaningful relationship among areas is explained.
- Overview is generated -> methods, applications, and possible learning routes are present.
- Discipline map opens -> one coherent entry appears instead of a five-zone sequence.
- Overview names unselected areas -> runtime receives no folder-creation instruction for them.
- Overview generation -> registered `encyclopedia` charter is assembled into a project-level prompt package and the selected native Agent CLI is launched; no HTTP model provider is called.
- CLI launch -> a normal session, run directory, prompt, and project-root `overview.md` output contract are recorded without inventing a sixth learning zone.
- Overview write -> the filesystem watcher emits `artifact-updated` with `zone: overview`.
- Key concepts and branches -> each selectable topic is emitted as a level-three heading; non-actionable visual subdivisions do not use H3.

## Frontend

- [US-07.1] Create flow opens -> explicit product-type choice is visible.
- [US-07.2] Advisor trigger renders -> it is icon-led at the upper-right of `项目形态` and opens with an incomplete form.
- [US-07.2] Advice succeeds -> multi-turn reply, recommendation, trade-off, and adopt action are visible.
- [US-07.3] Discipline map opens -> a Word-style heading table of contents precedes `学科总览`, and the zone timeline does not appear.
- [US-07.3] Overview generation starts -> the UI says it is opening the Agent CLI; after launch it says the visible terminal owns the run, without a fake duration estimate.
- System-learning project opens -> the five-zone experience remains available.
- [US-07.4] Sidebar renders -> discipline map is the clickable folder title and never an uncategorized child row.
- [US-07.5] Every H3 key concept or branch has one adjacent body action; no topic-card index or generic top action renders.
- [US-07.5] Inline topic-to-project action runs -> prefilled confirmation form opens.
- [US-07.6] Confirmation submits -> the standard create API receives no parent relation.
- [US-07.6] Map-origin project is created -> existing folder membership places it under the map-backed folder.
- [US-07.6] System-learning project row renders -> existing move-to-folder control remains available.
- Create modal at desktop or short viewport -> footer actions remain visible and the body scrolls independently.
- Discipline-map type selected -> system-learning-only motivation and level fields are hidden and absent from the create payload.
- [US-07.8, superseded by iteration 09] Historical quick-bridge UI test is replaced by direct summary and page-order coverage.

## Manual (optional; skipped when the maintainer requests automated-only validation)

- create a physics map and confirm that Newtonian mechanics, fluid mechanics,
  and thermodynamics appear in the overview but not as sidebar entries;
- inspect that the document begins with a Word-style table of contents rather than a card index;
- select the inline action beside fluid mechanics in the body and confirm that the editable form is prefilled;
- invoke the encyclopedia agent and confirm the selected native Agent CLI terminal opens visibly;
- confirm creation and verify the project directory is top-level;
- confirm the map title itself is a folder-level overview entry and is not duplicated under `未分类`;
- confirm the deep dive appears inside that map-backed folder and remains movable;
- reload the app and verify project type, overview, classification, and project status;
- inspect desktop and narrow layouts for the type chooser, advisor, overview,
  prefilled confirmation, sticky modal actions, and map-backed sidebar classification.

## Verification Commands

```powershell
go test ./backend-go/...
```

```powershell
Set-Location frontend
npm test -- --run
npm run build
```
