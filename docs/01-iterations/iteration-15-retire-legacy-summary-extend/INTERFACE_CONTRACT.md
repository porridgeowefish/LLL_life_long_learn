# Iteration 15 Interface Contract

## Removed interfaces

The server no longer registers:

```text
GET  /api/projects/{id}/summary/flashcards
POST /api/projects/{id}/summary/flashcards/grade
```

The browser exposes no Summary, Extend, or Knowledge Garden route. Existing
URLs fall through to the SPA's normal unknown-route behavior and do not revive
the retired feature.

## Retained interfaces

The active teacher workspace API, generated-artifact API, Sources API, Assets
API, and Explain/Practice compatibility readers are unchanged by this slice.

`summary` in provider reasoning frames, annotation Ask-AI records, and task
result metadata is unrelated to the retired Summary zone and remains valid.
