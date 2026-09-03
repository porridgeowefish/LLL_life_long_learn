# Acceptance Criteria

```text
A user can create a top-level learning project
A user can create a subproject under an existing project
Each created project includes Intro, Explain, Practice, Extend, and Summary structure
The project tree can show parent projects and child subprojects
Explain Agent can be invoked from the Explain zone
Invoking Explain Agent launches a real Claude Code terminal session
The backend records a session resource and append-only turns for that invocation
A user can send at least one follow-up into the same session
Follow-up output is appended in the panel and persisted in session history
Raw run files are stored separately from explain/output.md
summary/summary.md remains editable and is not automatically overwritten by runtime output
project-memory files exist and can store initial project learning memory
Reopening the app still allows the user to inspect the created project, its Explain output, and its recorded runs from disk
```
