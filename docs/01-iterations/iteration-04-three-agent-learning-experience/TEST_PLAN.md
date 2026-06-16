# Iteration 04 Test Plan

Status: active
Last reviewed: 2026-06-15

## Automated

- Agent registry loads new primitives and charter contract fragments.
- Every production agent prompt forbids ASCII/Unicode character diagrams and requires Mermaid for diagrams.
- Prompt assembly embeds project.md creation fields for every agent.
- Intro calibration does not repeat motivation, current level, target level, or completion standard already present in the project brief.
- Explain prompt contract rejects conversational learner-address language and diagnostic provenance in tutorial prose.
- Explain prompt contract confines first-principles reasoning to 2-4 final core-viewpoint sentences.
- Legacy tasks default to three-star subjective behavior.
- Multiple-choice answers compare as sets and lock after first submission.
- Latest Practice attempt restores submitted answers after refresh.
- Empty submitted attempts do not replace the latest meaningful answer set.
- Practice draft storage round-trips in-progress answers through `practice/draft.json`.
- Practice restore prefers matching disk draft/local draft and then overlays submitted attempt answers.
- Practice generation receives and enforces an exact structured question count.
- Practice evaluation prompts target one attempt without overwriting the task set.
- Evaluation artifacts contain an overall summary and per-question suggested answers.
- Answer keys return `403` through the file API.
- Growth events do not double-award on retry.
- Nested child projects resolve through their own slug.
- Frontend answer helpers handle text, boolean, and arrays.
- Explain summary anchors restore yellow highlights and fall back to quote
  matching when offsets drift.
- Summary deletion is permanent and legacy deleted records stay hidden.
- Project detail derives generated zone nodes from new and legacy artifacts.
- Summary Agent requires both structured flashcards and a human-readable
  review pack; cards emphasize concepts and core understanding.
- Summary flashcard storage reads the versioned envelope and legacy arrays,
  normalizes common generated variants, rejects duplicate IDs, and marks
  Summary generated from a valid card file.
- Flashcard UI supports filters, Markdown answers, flip controls, keyboard
  navigation, grading, and review progress.
- Run all Go tests, Vitest, and the production frontend build.

## Browser Smoke

- Intro renders assessment cards and confirmed child-project preview.
- Explain renders manifest navigation, page content, Mermaid, saved summaries,
  persistent highlights, batch copy, confirmation delete, and collapsible sidebar.
- Practice renders six question types, difficulty stars, immediate objective feedback, autosaved draft answers, preserved submitted answers, background AI summary, per-question suggested answers, and project growth.
- Regeneration returns to question-count selection and waits for a genuinely new task set.
- Summary renders a compact concept-card deck, flips without layout jumps,
  navigates by buttons and arrow keys, and exposes grading only after reveal.
- Existing projects without new files continue to render.
- The top bar and favicon use the cropped source lychee image without changing
  the existing Claude-style color tokens.
