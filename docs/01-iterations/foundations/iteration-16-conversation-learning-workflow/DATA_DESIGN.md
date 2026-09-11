# Iteration 16 Data Design

## Source parse output

```text
projects/<slug>/sources/<source-id>/revisions/<revision-id>/
  original/<uploaded-name>
  derived/content.md
  revision.json
```

`revision.json.derivedFiles` contains only the `content` file for a successful
parse. It is immutable with its revision.

## Teaching outline

`learning-scope.json` gains optional `teachingOutline` text. Map-origin creation
copies it from the selected catalog topic; standalone scope leaves it empty.

## Teacher usage

```text
projects/<slug>/conversation/usage.jsonl
```

One append-only JSON line per completed teacher response records response ID,
provider identifier when available, input tokens, output tokens, and timestamp.
No record is written when a provider supplies no usage values; values are never
estimated.
