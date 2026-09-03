# Iteration 05 Data Design

Status: active
Last reviewed: 2026-07-05

File remains the durable source of truth.

## Explain Files

```text
explain/manifest.json
explain/pages/NNN-slug.md
explain/confusions.json
```

No database migration is required.

## Manifest Page Metadata

New optional fields:

```text
updatedAt
revisionCount
lastUpdateReason
derivedFromPageId
structureRevision
```

Meaning:

```text
updatedAt          last durable update to this page entry
revisionCount      number of times the page has been revised after creation
lastUpdateReason   short human-readable reason for the latest update
derivedFromPageId  source page when this page is a new independent module
structureRevision  optional counter for structure-level rewrites
```

Readers must tolerate missing fields. Writers should populate them for new
Iteration 05 updates.

## Revision Storage

No page or manifest snapshot files are introduced in Iteration 05. Rewrites are
in-place edits to `explain/pages/*.md` and `explain/manifest.json`.

## Structure Rewrite Convention

The agent may revise the logical organization of Explain pages when the learner
question shows a better frame. Allowed changes:

```text
rename pages
reorder pages
split a mixed page
merge overlapping pages
create a module page
move sections between pages
```

Stable page IDs should be preserved for ordinary page rewrites. If a split or
merge makes that impossible, `lastUpdateReason` should explain the mapping in
plain language.

## Page Content Convention

When the learner contributes a hypothesis, challenge, or partial model, the
revised page should integrate it as durable learning content instead of burying
it in a transient answer. Recommended section labels are ordinary content
headings, not meta-chat:

```text
一个可保留的想法
这个想法成立的部分
它的边界
反例或修正
更稳的表述
```

These headings are suggestions, not schema keys. The artifact is still Markdown.
