# Iteration 17 — Test Plan

Status: planned
Owner: project maintainer
Last reviewed: 2026-09-11

## Backend Go tests

| area | file | cases |
|---|---|---|
| queue store | `conversation/store_test.go` | enqueue/edit/discard/promote projection; queue survives reload; supersede hides message and flips `ResponseForLearner`; steering marker recorded |
| queue service | `service/service_test.go` (fake gateway) | auto-advance starts head on completion; steer interrupts (interrupted status preserved) and starts new turn with steering appendix; steer on idle degenerates to promote; regenerate re-runs same learner trigger |
| websearch client | `teacher/internal/websearch/*_test.go` | httptest fake Zhipu API: result re-packing, error mapping, disabled-when-unconfigured |
| gateway tool loop | `gateway/gateway_test.go` | openai kind: tool_calls in assistant message + role=tool result round-trip via fake SSE; anthropic kind: tool_use/tool_result blocks; thinking suppressed when tools present on anthropic kind |
| OCR | `sources/internal/ocr/*_test.go` | httptest fake vision provider: base64 image payload, prompt contract, markdown output pass-through; failure → `ocr-failed` status |
| routes | `routes_learning_workspace_test.go` | queue/steer/regenerate/export endpoint contracts incl. 404/409 paths; existing contract-freeze suite stays green |
| config | `platform/config` | `webSearch` section parse/validate/env-key; missing section disables search |

## Frontend Vitest

| area | cases |
|---|---|
| `TeacherView` | input stays enabled while live; send while live calls queue endpoint and renders chip; queue edit/discard/steer actions wired; post-stream reattach picks up auto-advanced response; regenerate button only on last teacher message; copy button copies markdown source; export triggers download |
| frames | `search-started/completed/failed` render status line without disturbing markdown body |
| icons | shared icon component renders each glyph with aria-label |

## E2E (Playwright, existing fake-agent fixtures)

Queue → completion auto-advance → steer interrupt → regenerate last reply,
happy path only, using the deterministic fake provider.

## Manual smoke (learner-run, real keys)

1. GLM-4V OCR on a real 讲义截图：结构与公式检查。
2. 智谱 web-search 真实查询：来源链接出现在回答中。
3. 长对话刷新中排队的持久性；导出的 .md 打开检查。
4. 未配置 ocr/webSearch 时旧行为回归。

## Verification commands

```text
go test ./...
npm --prefix frontend run test
npm run archcheck
npm run check:fast
```
