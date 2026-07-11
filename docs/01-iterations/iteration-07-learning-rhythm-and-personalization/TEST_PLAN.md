# Iteration 07 Test Plan

Status: active

## Backend

- append/idempotency and legacy event decoding;
- global aggregation, filter, daily grouping, streaks, and 26/52-week ranges;
- client activity validation and server-owned timestamps;
- appearance load/save while preserving unrelated config keys;
- existing project progress response compatibility.

## Frontend

- rhythm loading/empty state, range switch, project filter, and day selection;
- effective-reading event is sent once after the threshold and interaction;
- theme application, persistence cache, migration from the removed system preference, and theme picker;
- GitHub links have accessible names and safe external-link attributes.

## Manual

- render home page at desktop and narrow widths;
- switch every theme and inspect navigation, forms, Markdown, heatmap, and focus rings;
- verify no theme flash on reload;
- confirm repository, Issue, and contribution links.
