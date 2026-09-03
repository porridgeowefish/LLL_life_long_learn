# Iteration 03 Delivery Notes

Status: in progress  
Owner: project maintainer  
Last reviewed: 2026-06-11  
Source of truth: current implementation audit against the iter-03 contract.

## Current State

iter-03 is partially implemented.

This directory is still the active target contract, but the codebase has not yet reached full delivery parity.

## Implemented In Code

Observed in `backend-go/` and `frontend/src/`:

```text
five-zone project skeleton
five-agent registry coverage
real agent invoke/session flow
Explain confusion CRUD
Practice generation entry + batch submit + evaluation read path
Summary flashcard read + grade flow
Markdown editor for learner-owned notes/summary
file-first persistence under projects/<slug>/
```

## Known Gaps Against Iter-03 Docs

```text
project creation still uses why/current/target/standard
  not currentAbility/targetAbility/difficulty/deliverables

Explain "统一提问" is not yet wired as one UI-driven invoke carrying sourceRefs
  current ConfusionPanel marks selected items as asked
  but does not submit them through AgentInvokePanel in one call

iter-03 API_CONTRACT.md still includes routes/shapes not fully matched by code
  example: practice evaluation is read via GET /practice/evaluation?attempt=N
  not the documented POST /practice/evaluations create endpoint
  example: summary flashcards support GET + POST grade, but not a generate endpoint

legacy/reference material still exists in the repo and should not be treated as active fallback
```

## Persistence Reality

Current persistence is:

```text
file-first
```

No active database layer is present in code.

Examples:

```text
explain/confusions.json
practice/tasks.json
practice/submissions/*.json
practice/evaluations/*.json + *.md
summary/flashcards.json
summary/flashcard-progress.json
runs/_index/*.json
```

## Recommended Next Cleanup

```text
finish iter-03 contract reconciliation before expanding scope
move any no-longer-needed legacy references toward archive ownership
update API_CONTRACT.md whenever backend routes change
keep DELIVERY_NOTES.md current so "implemented vs planned" is visible in-repo
```
