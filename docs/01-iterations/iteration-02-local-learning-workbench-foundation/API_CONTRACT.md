# API Contract

This iteration introduces the first project- and session-oriented contract surface.

The exact payload schema may evolve during implementation, but these resources define the intended slice boundary.

## GET /api/health

Purpose:
Return backend health and local runtime configuration summary.

Should include:

```json
{
  "ok": true,
  "workspace": "D:\\2_Study\\LLL",
  "claude": {
    "bin": "claude",
    "available": true
  }
}
```

## GET /api/projects

Purpose:
Return indexed top-level projects and subproject tree summaries.

## POST /api/projects

Purpose:
Create a new learning project with the default folder skeleton.

Request shape:

```json
{
  "title": "Recommender Systems",
  "slug": "recommender-systems",
  "parentProjectId": null
}
```

## POST /api/projects/:id/subprojects

Purpose:
Create a child learning project under an existing parent project.

## GET /api/projects/:id

Purpose:
Return one project, including zones, important file paths, and summary metadata.

## GET /api/projects/:id/tree

Purpose:
Return a project tree view optimized for the sidebar.

## GET /api/projects/:id/zones/:zone

Purpose:
Return one zone resource, including key files, artifact references, and latest session references.

## GET /api/agents

Purpose:
Return visible agent roles.

This iteration only requires production support for:

```text
Explain Agent
```

## POST /api/agents/:id/invoke

Purpose:
Invoke an agent on a project zone.

Request shape:

```json
{
  "projectId": "project-id",
  "zone": "Explain",
  "intent": "Explain user collaborative filtering for a learner with basic recommender systems context."
}
```

Behavior:

```text
resolve predecessor file paths
assemble prompt package
launch a real Claude Code terminal session
create a session resource
start event streaming for that session
```

## GET /api/sessions/:id

Purpose:
Return one session with status, turn list, linked run directory, and linked artifact references.

## POST /api/sessions/:id/follow-up

Purpose:
Append a follow-up question into the same session.

Request shape:

```json
{
  "text": "Compare collaborative filtering with content-based recommendation, but only give me a comparison framework."
}
```

Behavior:

```text
append a new user turn
write follow-up into the live Claude session when available
stream appended assistant output
do not replace prior turns
```

## POST /api/sessions/:id/cancel

Purpose:
Cancel a running Claude session.

## GET /api/events

Purpose:
Stream typed session and artifact events.

Expected event families:

```text
session-created
session-state
turn-created
terminal-output
artifact-updated
session-completed
session-failed
```

## Notes

This iteration does not require the full long-term target API surface.
It only needs enough contract to prove:

```text
project creation
subproject creation
agent invocation
session follow-up
artifact materialization
```
