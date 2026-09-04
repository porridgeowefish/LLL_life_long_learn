# Iteration 15 Data Design

## Retired writes

No new code writes either path:

```text
projects/<slug>/summary/**
projects/<slug>/extend/**
```

In particular, `summary/flashcards.json`, `summary/flashcard-progress.json`,
and `extend/flower.json` have no active reader, writer, or public API.

## Historical preservation

Existing files in these directories are immutable local history for this
iteration: no destructive migration, filesystem sweep, or automatic cleanup is
permitted. Iteration-13 migration must continue to complete when such files
exist, but it no longer inventories or promotes them as active input.

## Unaffected data

`assets/`, `sources/`, `conversation/`, `assistant-tasks/`, and Explain's
legacy compatibility files retain their current contracts.
