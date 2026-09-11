# Manual QA Scripts

Status: active
Owner: project maintainer
Last reviewed: 2026-09-11

One-off, human-run scripts that exercise a live backend. They are NOT part of
`npm run check` (see `tools/check/`); run them by hand when investigating a
specific area. Each script documents its prerequisites in its header.

| script | purpose |
|---|---|
| `qa_iteration13.py` | iteration-13 workspace regression sweep against a running server |
| `qa_long_conversation.py` | long-conversation pagination and context behavior |
| `qa_paginated_body.py` | paginated Body reader edge cases |
| `gen_infographic.py` | infographic generation helper for manual checks |

Moved here from `scripts/` in iteration 17 (see ADR-0019) so `scripts/` keeps
only operator entry points (Start/Stop/Install).
