# Database Design

This iteration remains file-first rather than database-first.

## Canonical Persistence

Use the filesystem as the durable source of truth.

Project skeleton:

```text
projects/
  <project-slug>/
    project.md
    state.json
    memory/
      project-memory.md
      project-state.json
    intro/
    explain/
      output.md
    practice/
    extend/
    summary/
      summary.md
    runs/
    assets/
    subprojects/
```

## Runtime Indexes

The backend may maintain in-memory indexes for:

```text
project tree
active sessions
session turns
artifact references
```

But these are caches only.

## Required Durable Files

This slice should write durable files for:

```text
project metadata
project memory
summary content
session prompt package
session stdout/stderr or transcript
curated explain output
```

## Follow-Up

If later iterations need a local embedded database for faster indexing, it should be introduced as an acceleration layer, not as a replacement for project files as the primary truth.
