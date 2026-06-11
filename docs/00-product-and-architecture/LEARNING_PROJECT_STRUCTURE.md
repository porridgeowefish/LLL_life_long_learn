# Learning Project Structure

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: this document defines the long-lived structure of LLL learning projects, subprojects, agent invocation, and memory growth.

## Intent

LLL should treat a learning task as a local project, not a single prompt run.

The project structure must support:

```text
folder-first organization
sidebar project tree navigation
fixed learning flow zones
direct Claude Code terminal invocation
multi-agent orchestration through file paths
editable summaries
long-term memory growth
subproject nesting
```

## Core Principle

One learning task equals one project folder.

The folder is the durable unit for:

```text
scope
context
agent inputs
agent outputs
practice artifacts
summary
memory
subprojects
```

LLL is not a chat container.
LLL is a local learning workbench organized around projects and files.

## Workspace Structure

Recommended top-level workspace layout:

```text
projects/
  <project-slug>/

memory/
  learner-profile.md
  learner-state.json
  learning-preferences.md

agents/
  registry/
  charters/

templates/
  projects/
  prompts/
```

Meaning:

```text
projects/ stores all learning projects and subprojects
memory/ stores global long-term learner memory
agents/ stores agent role definitions and behavior rules
templates/ stores reusable project skeletons and prompt templates
```

## Project Model

Each project should contain:

```text
one project overview
five fixed learning zones
one project memory area
one runs area
one assets area
one subprojects area
```

Recommended project structure:

```text
<project-slug>/
  project.md
  state.json
  memory/
    project-memory.md
    project-state.json
  intro/
    brief.md
    output.md
    notes.md
  explain/
    brief.md
    output.md
    notes.md
    map.mmd
  practice/
    brief.md
    tasks.md
    submissions/
    reflections.md
  extend/
    brief.md
    prompts.md
    relation-notes.md
    open-questions.md
  summary/
    summary.md
    next-steps.md
  assets/
  runs/
  subprojects/
```

## Required Files

`project.md`

Human-readable project home.
Should describe:

```text
topic
why this topic matters
current ability
target ability
completion standard
current phase
active subprojects
```

`state.json`

System-readable project state.
Should track:

```text
project id
title
status
active zone
last updated time
linked runs
linked assets
child project ids
```

`memory/project-memory.md`

Human-readable memory for this project.
Should capture:

```text
what the learner already understands
recurring confusion points
preferred explanation style
useful analogies
known gaps
```

`memory/project-state.json`

System-readable memory snapshot for orchestration and retrieval.

`summary/summary.md`

The canonical editable summary.
This file must remain learner-owned.
Agents may propose edits, but LLL should treat this file as personally editable knowledge, not a generated terminal dump.

## Learning Flow Zones

Each project uses five fixed zones.

### Intro

Purpose:

```text
create curiosity
activate prior knowledge
show why the topic matters
surface motivating questions
```

Expected outputs:

```text
interest hook
entry questions
relevance to the learner
minimal topic framing
```

### Explain

Purpose:

```text
deliver structured understanding
use MECE + first-principles decomposition
build concept maps and boundaries
```

Expected outputs:

```text
structured explanation
concept relationships
MECE breakdown
misconceptions
maps and diagrams
```

### Practice

Purpose:

```text
produce learning through constrained action
reduce dependence on AI during execution
make the learner generate output personally
```

Expected outputs:

```text
practice tasks
submission artifacts
self-reflection
minimal-AI or no-AI exercises
```

Rule:

```text
Practice agents should design work, constraints, and review criteria.
They should not directly complete the learner's work unless explicitly requested.
```

### Extend

Purpose:

```text
push the learner into higher-order thinking
focus on relations between knowledge points
support accumulation instead of direct answer delivery
```

Target relation types:

```text
causal relations
structural composition relations
priority or degree relations
isomorphic relations
```

Expected outputs:

```text
prompted comparisons
guided questions
relationship scaffolds
open-ended thought paths
```

Rule:

```text
Extend agents should not directly provide the final high-level conclusion when the goal is learner accumulation.
They should guide, constrain, and provoke.
```

### Summary

Purpose:

```text
capture what was actually learned
turn scattered outputs into reusable knowledge
support personal editing and later revision
```

Expected outputs:

```text
editable summary
what is understood
what remains unclear
next study steps
reusable expressions
```

Rule:

```text
Summary is the learner-facing canonical knowledge layer.
It must support direct editing in the LLL panel.
```

## Zone Dependency Graph

LLL should treat the five zones as a graph with predecessor nodes.

Default dependency graph:

```text
Intro -> Explain
Explain -> Practice
Explain -> Extend
Practice -> Extend
Intro + Explain + Practice + Extend -> Summary
```

Meaning:

```text
when an agent is invoked for a zone, LLL should resolve predecessor files first
the predecessor file paths become part of the runtime prompt context
the current zone behavior rules also become part of the runtime prompt
```

## Agent Model

Agents are visible roles in the LLL panel.

Each agent should have:

```text
identity
role charter
allowed project zones
behavior rules
expected outputs
target file paths
```

Recommended starter agents:

```text
Intro Agent
Explain Agent
Practice Agent
Extend Agent
Summary Agent
Memory Agent
```

## Agent Invocation Contract

When a user invokes an agent, the system behavior should be:

```text
1. Resolve the active project and current target zone
2. Resolve predecessor node file paths for the target zone
3. Build a prompt containing:
   - the selected agent identity
   - the agent charter
   - the predecessor file paths
   - the current task intent
   - the behavior rules for this zone
   - the output specification
   - the target output file paths
4. Launch a real Claude Code terminal session on the local machine
5. Auto-inject the prepared prompt into that Claude Code terminal
6. Allow the learner to observe the authentic Claude terminal feedback directly
7. Persist generated outputs into the project files
8. Index the outputs back into the LLL panel for reading, review, and editing
```

Important constraint:

```text
LLL does not replace the real Claude Code execution surface.
LLL launches and organizes it.
The terminal is the execution surface.
The panel is the observation and knowledge organization surface.
```

## Runs And Artifacts

`runs/` should store raw execution traces for replay and audit.

Recommended run structure:

```text
runs/
  <timestamp>-<agent-name>/
    prompt.md
    stdout.log
    stderr.log
    result.md
    run.json
```

Rule:

```text
raw runs are not the final knowledge layer
they are execution records
summary and project files remain the curated learning layer
```

## Subproject Rules

Subprojects allow deep study inside a broader parent topic.

Example:

```text
recommended systems/
  subprojects/
    user-collaborative-filtering/
```

Rules:

```text
each subproject is a full project with the same five-zone structure
subprojects inherit context from the parent project
subprojects may reference parent files as predecessor inputs
subproject summaries may be promoted back into the parent summary
parent projects should track active subprojects in project.md and state.json
```

Parent project meaning:

```text
map and organizing layer
```

Subproject meaning:

```text
focused deep-dive learning layer
```

## Memory System

LLL should keep memory at two levels.

Global learner memory:

```text
who the learner is
what they are trying to become
stable preferences
recurring learning patterns
```

Project memory:

```text
what is already learned in this topic
where confusion remains
which examples worked
which tasks were too easy or too hard
```

Memory should improve:

```text
task prompts
practice calibration
summary relevance
extension questions
next-step recommendations
```

Rule:

```text
memory should guide personalization, not silently override learner control
```

## Sidebar And Panel Model

Recommended sidebar structure:

```text
Project
  Overview
  Memory
  Intro
  Explain
  Practice
  Extend
  Summary
  Runs
  Assets
  Subprojects
```

Recommended main panel behavior:

```text
open files directly
show rendered output and raw file view
show agent actions for the current zone
show linked predecessor nodes
show editable summary content
```

## Out Of Scope For This Structure

This structure does not require:

```text
cloud deployment
user accounts
multi-tenant permissions
remote collaboration
complex message bus orchestration between agents
database-first project modeling
```

## Decision Summary

The LLL learning framework should be:

```text
project folders as the core unit
five learning zones as the fixed workflow
real Claude Code terminal invocation as execution
panel-based observation and editing as experience
file paths as the primary agent interface
memory growth as the personalization engine
subprojects as recursive deep-dive units
```
