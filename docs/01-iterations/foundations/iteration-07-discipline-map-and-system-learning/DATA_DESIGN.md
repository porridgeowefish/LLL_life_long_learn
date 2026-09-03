# Iteration 07 Data Design

Status: active  
Owner: project maintainer  
Last reviewed: 2026-07-14  
Source of truth: iteration 07 file/schema delta; implemented Go structs and persisted JSON supersede this document.

## Structure Overview

LLL remains file-first. Every new project is a flat top-level directory.
The existing folder layout remains the one navigation/classification structure.
A folder may bind one discipline-map slug as its clickable overview.

## Project State Delta

| Field | Type | Constraint | Change |
|---|---|---|---|
| `projectType` | enum string | missing legacy value decodes as `system-learning` | add |
| `activeZone` | enum string or omitted | system-learning only | modify |

No parent or child relation is stored. Folder placement does not enter prompt
assembly or change the token budget.

## Project Brief Shapes

`project.md` is also type-aware:

```text
discipline-map  -> 项目形态 + 总览目标 + 范围备注 + 备注
system-learning -> Why + Current ability + Target ability + Completion standard + Active Phase + Notes
```

The create client omits system-learning-only fields for a map. The skeleton
writer independently ignores those fields when `projectType=discipline-map`,
so the persisted brief cannot contradict the project state.

## Folder Layout Delta

`folders.json` keeps its existing folder array and adds one optional field:

| Field | Type | Meaning |
|---|---|---|
| `mapProjectSlug` | string or omitted | flat discipline-map project opened from this folder title |

The bound map slug is excluded from every `slugOrder`; `slugOrder` continues to
contain only classified system-learning project slugs. On read, the backend
creates a missing map folder or promotes a same-name existing folder. A stale
binding is cleared without deleting the folder or its membership.

## Flat Layouts

```text
projects/<map-slug>/
  state.json
  project.md
  overview.md
  memory/
  runs/
  assets/

projects/<learning-slug>/
  state.json
  project.md
  intro/
  explain/
  practice/
  extend/
  summary/
  progress/
  memory/
  runs/
  assets/
```

No `subprojects/` directory is created. All slugs are globally unique.

An encyclopedia run uses the existing `runs/<timestamp>-encyclopedia/` package.
Its package metadata sets `projectType: discipline-map`, omits `zoneName`, and
records `projectOutputPaths: ["overview.md"]`. This is a project-level Agent
contract, not a sixth zone. The CLI writes the final Markdown at the map root.

## Sync Points

| Reader / writer | Required behavior |
|---|---|
| state decoder | default missing project type to system learning |
| skeleton writer | always choose a flat top-level directory and write the project-type-specific `project.md` |
| index | expose every discovered project from flat storage |
| prompt/session routing | ignore frontend folder classification |
| project-level prompt assembly | validate `allowedProjectTypes`, record `overview.md`, and launch the selected native Agent CLI |
| artifact watcher | map root `overview.md` writes to `artifact-updated {zone: "overview"}` |
| folder store | reconcile discipline maps into existing folder objects |
| frontend store/sidebar | open map overview from folder title; render learning projects as child rows |
| project page | route by project type; derive a Word-style TOC and attach actions beside H3 body headings |

Historical prerequisite bridges were response-only and created no project,
folder, or relation. Iteration 09 removes that response shape in favor of the
generated `summary` field in `intro/prerequisite-assessment.json`.
