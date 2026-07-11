# Iteration 07 API Contract

Status: active

## Activity summary

`GET /api/activity?weeks=26&project=<optional-slug>`

`weeks` accepts `26` or `52`. Invalid values return `400`. An invalid project
slug returns `400`.

```json
{
  "summary": {
    "rangeStart": "2026-01-12",
    "rangeEnd": "2026-07-11",
    "activeDays": 46,
    "currentStreak": 5,
    "longestStreak": 12,
    "totalActions": 128,
    "totalGrowth": 368,
    "days": [{
      "date": "2026-07-11",
      "activity": 12,
      "growth": 14,
      "actions": 3,
      "events": []
    }]
  }
}
```

## Client activity

`POST /api/projects/{id}/activity`

Used only for client-observed meaningful reading. The backend owns the event
timestamp and applies an idempotent event id supplied by the client.

```json
{"id":"read:p001:2026-07-11","sourceId":"p001","title":"有效阅读","detail":"特征值的几何意义","activityDelta":2}
```

## Appearance

- `GET /api/settings/appearance`
- `PUT /api/settings/appearance`

```json
{"theme":"lychee-paper"}
```

Allowed themes: `lychee-paper`, `mountain-mist`, `wisteria-gray`, `night-ink`.

## Compatibility

Existing `GET /api/projects/{id}/progress` response remains unchanged. Older
events without `activityDelta`, `title`, or `detail` remain readable.
