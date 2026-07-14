# LLL PRD

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
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

A long-lived organizing project with one learner-facing `学科总览`. It presents
the discipline's major questions, research areas, methods, relationships,
boundaries, applications, and possible learning routes.

Topics named in the overview are possibilities, not project objects. They do
not enter the sidebar or filesystem until the learner creates a project.

### System-learning project

A committed deep-dive project using the existing five learning zones:

```text
Intro -> Explain -> Practice -> Extend -> Summary
```

It is a physically flat project regardless of whether creation started from a
map. When started from a map, existing folder membership places it beneath that
map-backed folder in navigation.

## Product Principles

```text
The learner explicitly chooses discipline map or system learning.
AI may hold a temporary clarification conversation, recommend, explain trade-offs, and prefill; it does not persist, silently choose, or create.
Every registered Agent, including the encyclopedia Agent, is explicitly launched through the selected native Agent CLI; lightweight AI helpers are the only direct-provider exception.
The overview represents the possibility space; the sidebar represents actual commitments.
The overview begins with Word-style heading navigation, then explains the same structure; each H3 key concept or branch has its learning action beside that body heading.
Only system-learning projects use the five-zone workflow.
Existing projects remain system-learning projects for compatibility.
Intro presents generated prerequisite-gap summaries directly, without a second action; project creation represents a deliberate longer-lived commitment.
All project directories are flat. Existing frontend folders are the only navigation/classification structure: a map opens from its folder title, while system-learning projects remain movable child rows. Folder placement never changes learning behavior or prompt context.
```

## Current Delivery Boundary

The two-type product model is implemented in iteration 07. Iteration 09 and
ADR-0008 own the current prerequisite-summary presentation. Current runnable
behavior and the iteration contracts are the delivery truth.
