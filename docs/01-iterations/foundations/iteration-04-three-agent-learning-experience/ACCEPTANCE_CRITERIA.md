# Iteration 04 Acceptance Criteria

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: historical black-box acceptance for iteration 04, with partial supersession noted below.

> Partially superseded by iteration 07 and ADR-0006 for prerequisite-gap actions.

```text
all agent prompts prohibit ASCII/Unicode character diagrams and use Mermaid for diagrams
Intro first writes 3-5 calibration questions including one application question to intro/survey.json
Intro calibration questions are answered in the rendered frontend page, not in the CLI/TUI
project.md creation fields are embedded in every generated prompt
Intro does not repeat project motivation, self-rated level, target, or completion standard
unknown answers never become claims of mastery
intro/assessment.json matches schema version 1
weak/missing prerequisite child creation required confirmation (historical; superseded in iteration 07)
created child project was directly navigable by its own slug (historical; no longer the default flow)

Explain uses the research-question framework
Explain output is a standalone tutorial, not an agent reply or learner diagnosis
learner profile and assessment provenance never appear in tutorial prose
first-principles reasoning appears only inside the final core viewpoints, never as its own section
complex explanations use manifest + ordered pages
follow-ups append kind=followup with parentPageId
old output.md remains readable

tasks support six declared types and 1-5 difficulty
five or more generated questions cover all five difficulty levels
generation and regeneration require an explicit question count
answer-key is absent from task responses and blocked by file API
objective checking locks the first answer and returns explanation
subjective evaluation remains batch-only
the learner may edit before one final submission; empty or duplicate final submissions are blocked
in-progress Practice answers persist to disk draft storage during answering
submitting preserves the full question set and answers and automatically starts background AI evaluation
refreshing Practice restores the latest submitted attempt from disk
refreshing Practice before submission restores matching disk draft answers
AI evaluation shows an overall summary plus feedback and a suggested answer for each subjective task
AI feedback is embedded in the corresponding question and never opens an Agent conversation

growth awards completion, objective correctness, and subjective evaluation
growth retries are idempotent
progress is project-local and has no ranking or streak behavior

Summary writes summary/flashcards.json and summary/review-pack.md
flashcards use the versioned file protocol and legacy arrays remain readable
cards test concepts, relationships, boundaries, misconceptions, and transfer
at least 60 percent of cards come from Explain core concepts
the Summary page provides a compact, keyboard-accessible flip-and-grade deck
```
