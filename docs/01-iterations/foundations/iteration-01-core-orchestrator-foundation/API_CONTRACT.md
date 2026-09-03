# API Contract

## GET /api/health

Purpose:
Return backend health and runtime configuration summary.

Response:

```json
{
  "ok": true,
  "claude": "claude",
  "workspace": "D:\\2_Study\\LLL"
}
```

## GET /api/tasks

Purpose:
Return all in-memory tasks.

## POST /api/tasks

Purpose:
Start a Claude Code task.

Request shape:

```json
{
  "title": "string",
  "prompt": "string",
  "cwd": "absolute path",
  "permissionMode": "default|acceptEdits|auto|plan|dontAsk"
}
```

## POST /api/tasks/:id/cancel

Purpose:
Cancel a running task.

## GET /api/events

Purpose:
Stream task lifecycle and log events over SSE.
