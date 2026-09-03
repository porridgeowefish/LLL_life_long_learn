# Iteration 05 Acceptance Criteria

Status: active
Last reviewed: 2026-07-05

```text
Explain follow-up no longer creates a new page by default
the experience encourages learner questioning instead of treating the first answer as final
the agent treats prior answers as revisable drafts when a follow-up exposes weakness
follow-up questions happen in the real terminal session, not a frontend question box
frontend does not add a new follow-up entry or update/create decision UI
parentPageId remains accepted as optional context but does not mean append a page
the agent chooses the relevant existing page from terminal context and manifest
the agent asks a short terminal clarification before writing when the target page is ambiguous

clarification, correction, example, caveat, and local deepening update the current page
new page creation is limited to independent concepts, methods, cases, or modules
questions that expose a better logical frame may restructure the Explain artifact
structure rewrites may rename, reorder, split, merge, or move sections across pages
new follow-up-created pages use kind=module rather than kind=followup
legacy kind=followup pages remain readable

page and manifest rewrites happen in place; Iteration 05 does not add snapshot storage
revised manifest entries include updatedAt, revisionCount, and lastUpdateReason
module pages include derivedFromPageId when created from a page-level follow-up
structure-rewritten pages include structureRevision or a lastUpdateReason explaining the rewrite

learner hypotheses are preserved when they are part of the question
AI responses distinguish useful partial insight, factual error, missing condition, and open uncertainty
the revised page reads as durable learning material, not a transcript of the exchange
the agent does not erase independent learner reasoning while correcting mistakes
the agent answers the learner's doubt directly before or while updating structure and pages
the artifact visibly improves through follow-up-driven answer iteration

Explain navigation avoids noisy one-off answer pages
Explain navigation reflects the best current logical structure, not the first generated outline
Practice and Summary source references continue to work with revised pages
all Go tests, frontend tests, production build, terminal smoke, and browser smoke pass
```
