# Iteration 16 Interface Contract

## Sources

`POST /api/projects/{id}/sources` retains its multipart shape. A supported file
with accepted parse disclosure returns `201` and a source in `processing`; its
source-processing task is internal. A ready revision exposes one derived file
with key `content` and media type `text/markdown`.

`GET /api/projects/{id}/sources/{sourceId}/revisions/{revisionId}/content`
returns the parsed Markdown only when that revision is ready. The Sources page
does not call it; the Teacher citation picker does.

## Usage

`GET /api/usage/teacher?page=<positive>&pageSize=<1..100>` returns:

```json
{"page":1,"pageSize":20,"total":1,"conversations":[{"projectSlug":"mapreduce","title":"MapReduce","conversationId":"conv_1","inputTokens":1200,"outputTokens":800,"turns":[{"responseId":"resp_1","occurredAt":"...","inputTokens":1200,"outputTokens":800}]}]}
```

## Navigation

The standalone frontend `/agents` page is retired. `/usage` is the
teacher-token page; compatibility Agent/session endpoints remain while legacy
callers still use them.
