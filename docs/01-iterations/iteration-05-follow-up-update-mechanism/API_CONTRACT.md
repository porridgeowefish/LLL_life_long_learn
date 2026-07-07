# Iteration 05 API Contract

Status: active
Last reviewed: 2026-07-05

## Follow-Up Surface

Iteration 05 does not add a frontend follow-up entry, chat box, or page-update
decision control.

The learner asks follow-up questions directly inside the real terminal session
opened or resumed for the Agent. The API contract is therefore mostly a prompt
and artifact contract:

```text
frontend -> opens/resumes terminal and renders files
terminal -> carries the learner's follow-up conversation
agent    -> updates explain/pages/*.md and explain/manifest.json using this policy
```

Existing endpoints remain valid:

```text
POST /api/agents/{id}/invoke
POST /api/projects/{id}/explain/resume
```

`POST /api/agents/{id}/invoke` may still accept legacy `parentPageId` from old
UI paths, but Iteration 05 does not require new request fields for follow-up.
`parentPageId`, when present, is only contextual guidance for the prompt; it is
not an instruction to append a page.

## Terminal Prompt Contract

The Explain Agent prompt must include standing rules for terminal follow-up:

```text
If the learner asks a clarification, correction, example, caveat, or local
deepening question, revise the relevant existing page.

Only create a new page when the learner introduces a distinct concept, method,
case, theorem, or module that deserves its own navigation entry.

If the learner's question exposes a better logical structure, reorganize the
artifact: update page titles, order, section boundaries, splits, merges, and the
manifest as needed. Do not preserve the old structure merely because it was the
first generated answer.

When the learner proposes a hypothesis or challenge, preserve the useful part
inside the revised page before correcting, bounding, or extending it.
```

Because the follow-up happens in terminal natural language, the agent decides
the target page from the active conversation and the existing manifest. If the
target is ambiguous, the agent should ask a short clarification in the terminal
before writing files.

## Structure Rewrites

A structure rewrite is allowed when a terminal follow-up shows that the current
Explain sequence is less useful than another organization. Examples:

```text
the learner identifies a missing distinction that should become the organizing axis
the learner's question reveals that page order should be prerequisites -> mechanism -> consequence
two pages are really one concept split by accident
one page mixes several independent concepts and should be split
the original explanation answered the topic, but not the learner's actual frame
```

When restructuring, the agent may:

```text
rename pages
change page order
split one page into multiple pages
merge nearby pages
create a module page
rewrite the manifest to match the new logical structure
```

The agent must keep source references valid as much as possible by preserving
stable page IDs for revised pages. If a page is split or merged, the manifest
should record the change with `lastUpdateReason`.

## Explain Manifest

`explain/manifest.json` remains the navigation source. Page entries may add:

```json
{
  "id": "p003",
  "title": "DFA and NFA equivalence",
  "file": "pages/003-dfa-nfa-equivalence.md",
  "kind": "core",
  "parentPageId": null,
  "order": 3,
  "updatedAt": "2026-07-05T12:00:00Z",
  "revisionCount": 2,
  "lastUpdateReason": "clarified learner challenge about subset construction",
  "derivedFromPageId": null,
  "structureRevision": 1
}
```

New writes should use:

```text
kind=core       original or revised main sequence page
kind=module     independent module created from a follow-up
kind=followup   legacy only; readable but not preferred for new writes
```

When a module is created from a page-level follow-up, set
`derivedFromPageId` to the source page.

`structureRevision` is optional and increments when a page participates in a
structure-level rewrite.

## Revision Storage

Iteration 05 does not add page or manifest snapshot storage. The Agent may
rewrite existing Explain pages and `explain/manifest.json` in place when the
terminal follow-up justifies it.

## Compatibility

- Existing manifests with `kind=followup` and `parentPageId` remain valid.
- Existing frontend code that sends only `parentPageId` should still work, but
  the prompt contract treats it as context rather than an unconditional append.
- Legacy `explain/output.md` fallback remains unchanged.
