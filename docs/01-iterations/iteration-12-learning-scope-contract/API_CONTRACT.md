# API Contract

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28

## Read Topic Boundaries

`GET /api/projects/{mapSlug}/discipline-topics`

Returns:

```json
{
  "schemaVersion": 1,
  "topics": [
    {
      "id": "classical-mechanics",
      "title": "经典力学",
      "chapterTitle": "力与运动",
      "goal": "解释宏观低速物体的运动规律",
      "inScope": ["牛顿运动定律"],
      "outOfScope": ["热力学"],
      "prerequisites": ["向量"],
      "ownedConcepts": ["惯性参考系"],
      "reusedConcepts": ["微积分"]
    }
  ],
  "updatedAt": "RFC 3339"
}
```

Missing catalogs read as an empty schema-v1 catalog. Malformed or semantically
invalid catalogs return `500 read_discipline_topics`.

## Create A Scoped Deep Dive

`POST /api/projects`

The normal system-learning body may add:

```json
{
  "scopeSource": {
    "type": "discipline-map",
    "mapSlug": "physics",
    "topicId": "classical-mechanics"
  }
}
```

The client does not send the boundary content. The backend resolves it from the
canonical map artifact and writes the snapshot in the same creation operation.

Errors:

| Status | Code |
|---|---|
| 400 | `scope_source_requires_system_learning` |
| 400 | `invalid_scope_source` |
| 400 | `discipline_topics_unavailable` |
| 400 | `discipline_topic_not_found` |

`learning-plan.json` items may include optional `topicId`; title-only legacy
items remain valid.
