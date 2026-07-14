# Iteration 09: Learning Lifecycle Cleanup

Status: active delivery slice
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: iteration 09 scope and delivery boundary.

## Goal

Clean up two obsolete lifecycle surfaces: allow permanent deletion of a
system-learning project after one confirmation, and replace the on-demand
prerequisite supplement with generated gap summaries.

The previously recorded discipline-protocol and local-learning-lab ideas remain
discovery-only in [BACKLOG.md](./BACKLOG.md); they are not part of this delivery
slice.

## Scope

```text
sidebar project action and one destructive confirmation
DELETE /api/projects/{id}
whole project-directory removal
folder-reference, project-index, and in-memory session cleanup
browser-local project draft cleanup
active-session race protection
generated prerequisite summaries in intro/assessment.json
Intro order: generated content -> diagnosis -> survey form
removal of the quick-supplement button and endpoint
```

## Documentation Impact

Project deletion itself requires no ADR. The prerequisite workflow change is
owned by `ADR-0008` and supersedes the on-demand bridge in `ADR-0006`.

Long-lived facts synchronized in this slice:

- `API_CONTRACT_STRATEGY.md`
- `DATA_MODEL.md`
- `PRD.md`
- `DOMAIN_MODEL.md`
- `LEARNING_PROJECT_STRUCTURE.md`
- `AGENT_ARCHITECTURE.md`
- `SYSTEM_ARCHITECTURE.md`
- `MVP_ROADMAP.md`
- `docs/01-iterations/README.md`
