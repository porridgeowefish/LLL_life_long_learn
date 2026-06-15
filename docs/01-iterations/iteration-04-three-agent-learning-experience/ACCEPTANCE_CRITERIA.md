# Iteration 04 Acceptance Criteria

Status: active
Last reviewed: 2026-06-15

```text
Intro charter asks 3-5 calibration questions and one application question
unknown answers never become claims of mastery
intro/assessment.json matches schema version 1
weak/missing prerequisite cards require confirmation before child creation
created child project is directly navigable by its own slug

Explain uses the research-question framework
Explain output is a standalone tutorial, not an agent reply or learner diagnosis
learner profile and assessment provenance never appear in tutorial prose
first-principles reasoning appears only inside the final core viewpoints, never as its own section
complex explanations use manifest + ordered pages
follow-ups append kind=followup with parentPageId
old output.md remains readable

tasks support six declared types and 1-5 difficulty
five or more generated questions cover all five difficulty levels
answer-key is absent from task responses and blocked by file API
objective checking locks the first answer and returns explanation
subjective evaluation remains batch-only

growth awards completion, objective correctness, and subjective evaluation
growth retries are idempotent
progress is project-local and has no ranking or streak behavior
```
