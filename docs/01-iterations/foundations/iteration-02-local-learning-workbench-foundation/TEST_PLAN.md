# Test Plan

## Backend

```text
Create a top-level project and verify the default folder skeleton is written to disk
Create a subproject and verify it is placed under the parent project's subprojects/ directory
Call GET /api/projects and confirm project tree indexing works
Invoke Explain Agent and confirm a real Claude Code terminal session is launched
Confirm predecessor file paths are included in the prompt package saved to runs/
Confirm session creation produces append-only turn records
Send follow-up to the same session and confirm the new turn is appended instead of replacing earlier content
Confirm explain/output.md is written separately from raw run artifacts
Confirm summary/summary.md is not blindly overwritten by runtime output
Confirm project-memory files can be created and updated
```

## Frontend

```text
Load the page and confirm the project tree renders top-level projects and subprojects
Open a project and confirm all five zones are visible
Invoke Explain Agent from the Explain zone and confirm the UI reflects a live session
Confirm the main learning panel stays the dominant layout region
Confirm follow-up history is appended in order
Confirm users can review both raw session history and curated Explain output
Confirm Summary remains editable
```

## Good Test Rule

Good tests for this iteration should verify external behavior:

```text
what files exist
what session states are exposed
what turn history is returned
what artifacts are written
what users can see and continue
```

Avoid tests that depend on:

```text
internal map shapes
private helper names
implementation-specific buffering details
```
