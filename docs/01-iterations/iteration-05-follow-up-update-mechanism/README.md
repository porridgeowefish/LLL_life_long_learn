# Iteration 05: Follow-Up Update Mechanism

Status: active
Owner: project maintainer
Last reviewed: 2026-07-05
Source of truth: this directory defines the fifth LLL delivery slice.

Editable alignment handoff: [ALIGNMENT.md](./ALIGNMENT.md)

## Goal

Make repeated learner questions improve the existing learning artifact instead
of scattering answers into extra pages.

The overall value orientation is:

```text
encourage learner questioning
treat answers as improvable drafts
use follow-up dialogue to iteratively optimize both content and structure
```

Current behavior treats an Explain follow-up as a new page by default. That
makes the artifact grow like a chat transcript: original page, doubt, answer,
another page. Iteration 05 changes the default:

```text
clarify / correct / deepen existing material -> update the current page
new independent knowledge module           -> create a new page
better logical structure emerges           -> reorganize pages and manifest
learner's own hypothesis or challenge       -> preserve it explicitly, then respond
```

The desired experience is a living study artifact, not a pile of appended
replies. The agent should be willing to answer the learner's doubt directly and
then improve the artifact's structure when the doubt reveals a better way to
organize the knowledge.

## Included

```text
Explain follow-up update policy
current-page revision instead of default follow-up append
manifest metadata for revised pages
structure-level revision of page order, titles, splits, and merges
terminal follow-up behavior rules for the existing Agent conversation
prompt contract that preserves learner hypotheses, objections, and partial models
compatibility for existing follow-up pages and parentPageId requests
tests for update/create/restructure decision boundaries
```

## Excluded

```text
collaborative multi-user editing
automatic merging across unrelated pages
database-first document revision storage
full rich-text editor
agent memory redesign
new frontend follow-up entry or chat box
page or manifest revision snapshot storage
Practice answer evaluation changes
Summary flashcard protocol changes
```

## Key Decisions

- A follow-up is not a page type by default; it is an update intent.
- Follow-up questions happen in the real terminal session. The frontend should
  not add a separate follow-up entry, chat box, or decision UI for this slice.
- Existing pages remain the primary artifact unless the follow-up introduces a
  distinct concept, method, theorem, case study, or module that deserves its own
  navigation entry.
- If the learner's question reveals that the current explanation is organized
  around the wrong distinction, wrong order, or weak conceptual frame, the agent
  should revise the structure instead of only patching local text.
- The agent must not erase the learner's independent thinking. It should retain
  useful hypotheses, label uncertainty, and explain where the idea works or
  breaks.
- Page rewrites and structure rewrites are allowed in-place this iteration; no
  revision snapshot storage is added.
- Legacy `kind=followup` pages stay readable. New writes should use revised
  core pages or `kind=module` pages instead.

## Product Principle

The system should behave less like "standard answer delivery" and more like a
thinking partner:

```text
invite the learner to question the answer
make answer iteration a normal workflow, not an exception
do not flatten the learner's question into a canned answer
do not punish speculative reasoning
separate factual correction from judgment of the learner
turn good partial ideas into named anchors inside the page
reorganize the explanation when the learner discovers a better frame
show why an idea is promising, limited, or wrong
```
