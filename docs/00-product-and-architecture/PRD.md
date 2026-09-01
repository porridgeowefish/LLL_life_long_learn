# LLL PRD

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: long-lived product intent and product-object boundaries for LLL.

## Problem

Serious learning with AI fails in two different ways:

```text
learning context, execution, notes, and review are scattered across tools
broad disciplines and focused learning goals are forced into the same content shape
```

The first causes weak continuity. The second causes premature depth: a learner
who needs a field map receives a tutorial, while a learner ready to build skill
needs a committed learning workflow.

## Product Goal

LLL is a local-first learning workbench that:

```text
keeps learning context and artifacts in durable local project folders
lets the learner choose the product shape instead of hiding that decision in AI inference
uses AI as an advisor and controlled study operator, not an autonomous product owner
supports broad orientation and focused skill-building without conflating them
keeps execution inspectable through real local AI runtime sessions
```

## Primary User

The primary user is a power learner who is comfortable with local files and
terminal-based AI runtimes and wants a durable thinking environment rather than
a disposable chat answer.

## Product Objects

### Discipline map

A long-lived organizing project with learner-facing `学科总览` and `学习计划` views. It presents
the discipline's major questions, research areas, methods, relationships,
boundaries, applications, a multi-level chapter/topic architecture, and possible learning routes.

In the sidebar it is shown as an explicit `××学科总览` row inside its bound
discipline folder. The folder title only expands or collapses the folder.

Topics named in the overview are possibilities, not project objects. They do
not enter the sidebar or filesystem until the learner creates a project.

### System-learning project

A committed deep-dive learning unit. Current delivered code uses the five
legacy learning zones. The accepted iteration-13 target replaces that active
presentation with one teacher conversation plus editable assets and source
material.

It is a physically flat project regardless of whether creation started from a
map. When started from a map, existing folder membership places it beneath that
map-backed folder in navigation.

Every system-learning project owns one explicit content boundary. A map-origin
project snapshots its selected topic's planned boundary; a standalone project
starts with a draft boundary. In the iteration-13 target, this boundary guides
the teacher and assistant without mechanically rejecting natural cross-topic
questions.

## Product Principles

```text
The learner explicitly chooses discipline map or system learning.
AI may hold a temporary clarification conversation, recommend, explain trade-offs, and prefill; it does not persist, silently choose, or create.
Every registered Agent, including the encyclopedia Agent, is explicitly launched through the selected native Agent CLI; lightweight AI helpers are the only direct-provider exception.
The overview represents the possibility space; the sidebar represents actual commitments.
The encyclopedia Agent plans the whole architecture before writing; the overview uses H3 major chapters and H4 learnable topics. A separate top-level learning-plan view lets the learner choose topics, own their order, and check off one task at a time; AI does not choose the sequence.
Each H4 topic also declares what it owns, reuses, requires, and excludes. The backend snapshots that canonical boundary when the learner confirms a deep dive.
Each new-contract H4 topic has its learning action beside that body heading; legacy H3-only overviews remain actionable until regenerated.
Only system-learning projects use the currently delivered five-zone workflow.
After iteration-13 migration they retain their type but use the conversation-first workspace.
Existing projects remain system-learning projects and migrate compatibly.
Intro presents generated prerequisite-gap summaries directly, without a second action; project creation represents a deliberate longer-lived commitment.
Intro calibrates readiness, depth, examples, scaffolding, and practice difficulty. It may finalize a standalone draft scope but cannot broaden a ready map-origin scope.
All project directories are flat. Existing frontend folders are the only navigation/classification structure: a map opens from its folder title, while system-learning projects remain movable child rows. Folder placement never changes learning behavior or prompt context.
```

## Current Delivery Boundary

The two-type product model is implemented in iteration 07. Iteration 09 and
ADR-0008 own the current prerequisite-summary presentation. Current runnable
behavior and the iteration contracts are the delivery truth.

Iteration 13 is the accepted next product transition. Its target has `教师` as
one low-latency API conversation, `资产` as versioned editable accumulated
learning, and `资料` as explicit private source material. A visible native-CLI
assistant performs approved substantial work asynchronously. The five teaching
roles are invisible methods in one prompt, not product modules. Until iteration
13 is implemented and its migration gate passes, the current five-zone runtime
remains the executable experience.
