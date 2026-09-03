# Iteration 08: Learning Rhythm And Personalization

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: this directory defines the eighth LLL delivery slice.

## Goal

Turn the home page into a learner-facing history surface and add calm,
accessible appearance choices plus clear open-source collaboration entry points.

## Included

- one append-only project learning event stream with separate activity and growth deltas;
- global and per-project 26/52-week rhythm aggregation;
- active days, current/longest streak, actions, growth, and day drill-down;
- event capture for agent invocation/follow-up, Ask-AI, Practice, flashcards, and effective reading;
- backward-compatible reuse of existing Practice growth events;
- four appearance choices: lychee paper, mountain mist, wisteria gray, night ink;
- semantic theme tokens, immediate preview, durable config, and no startup flash;
- GitHub repository entry plus Issue and contribution links.

## Excluded

- memory redesign or automatic memory updates;
- leaderboards, social comparison, achievements, or punitive streaks;
- precise time tracking or background surveillance;
- discipline thinking protocols and local code execution (iteration 09 discovery).

## Product Decisions

- activity measures investment; growth measures verifiable learning outcomes;
- automated agent file generation awards neither activity nor growth by itself;
- the learner's explicit invocation is an activity, while Practice keeps its existing growth policy;
- the accepted visual handoff is `frontend-designs/learning-overview-activity.html`;
- `progress/events.jsonl` remains the durable per-project source and is extended compatibly;
- themes change semantic roles, never component-local hard-coded colors.
