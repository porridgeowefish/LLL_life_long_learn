# ADR-0008 Generated Prerequisite Gap Summaries

Status: accepted
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: current prerequisite-gap presentation and generation boundary.

## Context

ADR-0006 replaced prerequisite child projects with an on-demand inline AI
bridge. After the subagent-oriented learning flow was removed, the remaining
`快速补充` action became a dead-end interaction: the diagnosis already knew the
gap, but the learner had to trigger another request to understand what it meant.

The Intro survey is an input surface and must remain the final section of the
Intro page. Placing generated diagnosis after it made the page read as if the
form were not the end of the calibration flow.

## Decision

`intro/assessment.json` owns a concise generated `summary` for every
prerequisite item. The summary explains in one or two sentences what the
knowledge is and, for weak or missing items, which layer of understanding is
currently absent. It does not become a tutorial or a second assessment.

The frontend renders the summary directly with impact and evidence. It exposes
no supplement button and makes no follow-up prerequisite API call.

Intro order is fixed:

```text
generated Intro content
prerequisite diagnosis
survey form
```

Legacy assessments without `summary` remain readable by using `impact` as the
display fallback.

## Consequences

- `POST /api/projects/{id}/prerequisite-bridge` is removed.
- The Intro Agent must generate concise summaries in the initial assessment.
- Explain content naturally supplies necessary background without exposing
  learner-diagnosis language.
- A prerequisite gap still does not create or imply a separate project.

## Supersedes

This ADR supersedes ADR-0006's on-demand inline bridge decision. ADR-0006's
project-commitment boundary remains historical context and is still respected.

