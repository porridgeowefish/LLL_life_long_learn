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
Summary Agent structured flashcards aligned with backend storage and frontend rendering
concept-focused flashcard deck with filters, keyboard navigation, Markdown, and grading
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

Project-context correction on 2026-06-15:

```text
project.md                    embedded in every generated prompt
package.json                  records the project brief path
Intro calibration             skips project-creation fields already answered
legacy projects               remain runnable when project.md is absent
```

Shared diagram contract on 2026-06-15:

```text
all agent prompts             prohibit ASCII/Unicode character diagrams
diagram output                uses fenced Mermaid blocks
code, formulas, tables        remain allowed
```

Practice feedback correction on 2026-06-15:

```text
submitted answers             persist and restore from the latest disk attempt
AI evaluation                starts automatically after batch submission
evaluation summary           shown above the submitted question set
subjective question feedback includes a direct suggested answer
failed evaluation launch     keeps answers and exposes a retry action
```

Practice workflow correction on 2026-06-15:

```text
question generation          requires an exact structured count
new question set             returns to count selection before generation
attempt creation             deferred until the learner checks or submits work
final submission             allowed once only after every question is complete
submitted questions/answers  persist and remain visible after refresh
empty legacy attempts        cannot replace the latest meaningful submission
AI evaluation                runs headlessly in the background
AI feedback                  renders inside each corresponding question
manual AI invoke button      removed
```

Practice draft persistence correction on 2026-06-16:

```text
practice/draft.json          added as durable in-progress answer storage
draft API                    GET/PUT endpoints added for autosave restore
frontend restore             reads disk draft, local draft, then overlays submitted attempt answers
empty legacy submissions     no longer overwrite non-empty matching drafts
submitted answers            restored from attempt 2 for the lexer project instead of showing empty cards
```

Validation:

```text
go test ./...                 passed
npm run test                  passed (15 files, 34 tests)
npm run build                 passed
git diff --check              passed
browser restore              selected meaningful attempt 2 instead of empty attempt 4
browser submitted view       original answers, overall feedback, and per-question answers visible
browser regeneration         returned to explicit question-count selection
browser console              no warnings or errors
```

Validation:

```text
go test ./...                 passed
npm run test                  passed (13 files, 31 tests)
npm run build                 passed
git diff --check              passed
browser smoke                restored attempt 2 answers and post-submit question navigation
browser console              no warnings or errors
```

Summary alignment correction on 2026-06-15:

```text
summary/flashcards.json      required as the frontend card source of truth
summary/review-pack.md       retained as concise human-readable review material
legacy flashcard arrays      remain readable
new flashcard writes         use the versioned envelope
card content                 prioritizes concepts and core understanding
Summary generated state      recognizes a valid flashcards.json file
```

Validation:

```text
go test ./...                 passed
npm run test                  passed (14 files, 33 tests)
npm run build                 passed
git diff --check              passed
browser smoke                9 cards loaded; flip and arrow navigation passed
browser console              no warnings or errors
```

Summary flashcard reader correction on 2026-06-16:

```text
flashcard reader             accepts canonical envelope, legacy arrays, fenced JSON, and common generated aliases
frontend DTO                 normalized back to id/front/back/category/sourceRefs/generatedReason
Summary generated state      uses the same tolerant protocol family
Summary prompt               now forbids schema/key aliases, fences, trailing commas, comments, and prose in flashcards.json
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
