# Iteration 10 Data Design

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: iteration 10 persisted-data effects.

No schema migration is required.

- `extend/flower.json` remains the canonical normalized knowledge-flower file and preserves newline characters inside JSON strings.
- `progress/events.jsonl` may now contain `sourceType: "extend-flower"` events with `activityDelta: 1` and zero growth value.
- The event ID derives from the saved file bytes, so the same byte representation is awarded once per project across the append-only stream.
- The internal theme preference ID `wisteria-gray` remains readable for compatibility; its learner-facing name and token values change to the sky-blue presentation.
