# Iteration 09 Data Design

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15

Project deletion adds no schema. Intro prerequisite items add the optional
`summary` compatibility field:

```json
{
  "summary": "用 1-2 句话介绍知识本身和当前缺口"
}
```

New Agent output must write it. Existing records without it remain readable by
falling back to `impact`.

## Deletion boundary

For project slug `<slug>`, deletion removes:

```text
projects/<slug>/
  state.json and project.md
  Intro/Explain/Practice/Extend/Summary artifacts
  memory and progress files
  runs and session index files
```

Associated references are also removed from:

```text
folders.json slugOrder and mapProjectSlug references
the in-memory project index
the in-memory session registry
known browser localStorage draft keys for the slug
```

Unrelated folders, folder members, projects, sessions, and browser drafts are
preserved. A folder whose map project is deleted becomes an ordinary folder.
