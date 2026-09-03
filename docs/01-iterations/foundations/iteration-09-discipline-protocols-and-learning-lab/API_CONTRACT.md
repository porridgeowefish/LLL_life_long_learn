# Iteration 09 API Contract

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15

## Delete project

```http
DELETE /api/projects/{id}
```

The client must obtain one explicit learner confirmation before calling this
endpoint. The endpoint itself is non-interactive.

Success (`200`):

```json
{
  "deleted": true,
  "projectId": "linear-algebra"
}
```

| Status | Error | Meaning |
|---:|---|---|
| `400` | `invalid slug` | `id` is not a valid project slug |
| `404` | `project not found` | canonical project state does not exist |
| `409` | `project_has_active_session` | an Agent session may still write project files |
| `500` | implementation message | filesystem or global-reference cleanup failed |

Deletion is permanent. The canonical project directory is the content boundary;
the server also prunes workspace-global folder references and in-memory session
metadata.

## Removed prerequisite helper

Iteration 09 removes `POST /api/projects/{id}/prerequisite-bridge`. Concise gap
content is part of the generated `intro/assessment.json` artifact instead of a
separate HTTP interaction.
