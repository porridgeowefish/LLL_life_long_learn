# Iteration 04 Delivery Notes

Status: implemented
Last reviewed: 2026-06-15

Implemented:

```text
three refactored charters and two new primitives
Explain tutorial artifact contract: neutral textbook prose, no learner-diagnosis narration
first-principles primitive restricted to final core-viewpoint compression
intro assessment UI and confirmed prerequisite child-project creation
explain manifest/page reader and follow-up invocation context
six practice question types, answer-key isolation, attempts, objective checking
project-local append-only growth events and summary endpoint
legacy Explain and Practice compatibility
compact dashboard session rows using the real session DTO
saved Explain summaries with persistent yellow highlights, copy, hard delete, and collapsible sidebar
derived generated-zone status for green timeline nodes
source-faithful lychee SVG in the top bar and favicon without changing the Claude-style palette
```

Validation completed on 2026-06-14:

```text
go test ./...                 passed
npm run test                  passed (10 files, 26 tests)
npm run build                 passed
browser smoke: Intro          passed
browser smoke: Explain        passed
browser smoke: Practice       passed
answer-key file isolation     passed (HTTP 403)
progress API/disk consistency passed
```

The browser smoke used a temporary fixture project and did not invoke Claude.

Prompt-contract correction validated on 2026-06-15:

```text
go test ./...                 passed
Explain production prompt     legacy first-principles contract absent
Explain artifact contract     tutorial voice and silent context use present
```

Frontend close-out validated on 2026-06-15:

```text
go test ./...                 passed
npm run test                  passed (11 files, 29 tests)
npm run build                 passed
git diff --check              passed
browser smoke: Explain nav    passed
browser smoke: summary panel  passed, including persisted collapse
browser smoke: zone timeline  generated nodes green
lychee asset                  cropped source pixels, transparent background, proportional 30px render
```
