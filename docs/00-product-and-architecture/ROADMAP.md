# LLL Roadmap

Status: draft
Owner: project maintainer
Last reviewed: 2026-06-09
Source of truth: which learning agents ship when, and which primitives each one owns.

## Iteration Status

```text
iter-02      Go backend + Five-zone model + Explain Agent + v2 frontend  ✓
iter-02.1    Mock alignment + UX repair + reasoning-primitive scaffolding ✓
iter-02.2    Deepthink replacement + reasoning-primitive library shipped  ✓
iter-03      Intro / Practice / Extend / Summary agents (planned)
iter-04      PTY-backed real session continuity (planned)
iter-05      Memory auto-update layer (planned)
```

## Reasoning Primitive Ownership

Every shipped primitive has exactly one owning agent today. Future agents take ownership of the primitives already written for them in iter-02.2; they should not require new primitives unless a genuine new mechanism appears.

```text
┌──────────┬─────────────────────────────────────┬────────────────────────┐
│ Agent    │ Required primitives                 │ Status                 │
├──────────┼─────────────────────────────────────┼────────────────────────┤
│ Explain  │ mece_decompose, first_principles,   │ SHIPPED (iter-02)      │
│          │ concept_graph, misconception,        │                        │
│          │ boundary_map                        │                        │
│          │ + optional: analogy                 │                        │
├──────────┼─────────────────────────────────────┼────────────────────────┤
│ Intro    │ knowledge_anchor, boundary_map      │ primitive ready,       │
│          │                                     │ agent pending iter-03  │
├──────────┼─────────────────────────────────────┼────────────────────────┤
│ Practice │ transfer                            │ primitive ready,       │
│          │                                     │ agent pending iter-03  │
├──────────┼─────────────────────────────────────┼────────────────────────┤
│ Extend   │ critical_thinking                   │ primitive ready,       │
│          │                                     │ agent pending iter-03  │
├──────────┼─────────────────────────────────────┼────────────────────────┤
│ Summary  │ review_pack                         │ primitive ready,       │
│          │                                     │ agent pending iter-03  │
└──────────┴─────────────────────────────────────┴────────────────────────┘
```

## iter-03 Plan: Four Remaining Agents

The four agents below were scoped out of iter-02.2 to keep the deepthink-replacement initiative focused. Each is now a relatively small delivery because its primitive library is already in place.

```text
Intro Agent
  Zone:           Intro
  User story:     spark curiosity + activate prior knowledge
  Required:       knowledge_anchor, boundary_map
  Output target:  intro/output.md
  Output contract:
    - Knowledge anchors (2-4 statements)
    - Relevance hook (why this topic matters)
    - Three entry questions
    - Minimal topic framing
    - Boundary map (IN / ADJACENT / PREREQUISITE)
  Estimated:      ~1 day (charter + registry entry + smoke test)

Practice Agent
  Zone:           Practice
  User story:     generate a 1-10 question transfer quiz (default 5)
  Required:       transfer
  Output target:  practice/tasks.json
  Output contract:
    - JSON quiz: { tasks: [{id, type, question}], generatedAt }
    - Each question is a transfer task (scenario + constraints + self-check)
    - No answers / solutions (P-02)
    - 1-10 questions, default 5 (P-01)
  Estimated:      ~1 day

Extend Agent
  Zone:           Extend
  User story:     probe the topic with counterfactuals + relational reasoning
  Required:       critical_thinking
  Output target:  extend/relation-notes.md
  Output contract:
    - Counterfactual cascade
    - Relational mapping (sibling + competitor)
    - Failure mode at the edge
  Estimated:      ~1 day

Summary Agent
  Zone:           Summary
  User story:     turn scattered outputs into reusable, reviewable knowledge
  Required:       review_pack
  Output target:  summary/summary.md (proposal only — learner-owned) +
                  summary/review-pack.md (review pack)
  Output contract:
    - Synthesised summary of prior zones
    - Feynman checklist (3-5 items)
    - Spaced-repetition cards (5-10)
    - Review cadence recommendation
  Constraint:     summary/summary.md is learner-owned; agent may only
                  propose edits via a structured "suggested edit" block,
                  not overwrite the file.
  Estimated:      ~1.5 days
```

## iter-04+ Plan: Real Session Continuity

```text
PTY-backed follow-ups
  Replace the current re-launch follow-up pattern with a persistent
  PTY session (Windows conpty + Go pty). Follow-ups within a session
  share Claude's in-memory state.

Disk-resident session index
  runs/_index/*.json is rebuilt on startup so server restarts recover
  active sessions.

Cross-platform terminal launch
  Linux xterm and macOS Terminal.app equivalents for the
  CREATE_NEW_CONSOLE pattern introduced in iter-02.1.
```

## iter-05+ Plan: Memory Auto-Update

```text
Memory agent
  A read-write memory layer that proposes updates to project-memory.md
  and learner-profile.md after meaningful runs. Still learner-owned —
  updates go through the same kind of structured proposal the Summary
  Agent uses for summary/summary.md.
```

## Out of Scope (Permanent)

```text
Multi-user support (LLL is a local tool)
Cloud sync (files stay on disk)
Database-first project model (filesystem is the source of truth)
Message bus between agents (they coordinate via files)
```

## Verification

```text
go test ./backend-go/...
  agentregistry package asserts every declared primitive exists
  promptassembly package asserts prompt expansion works
  coverage_test asserts the 11-mechanism table is fully covered

Manual smoke test
  1. go run ./backend-go/cmd/lll
  2. Open a project, invoke Explain Agent
  3. Verify projects/<slug>/runs/<ts>-explain/prompt.md contains
     # User Story + # Reasoning Primitives blocks with primitive bodies
     inlined
  4. Verify projects/<slug>/explain/output.md has First Principles,
     MECE Decomposition, Concept Graph, Misconceptions, Boundary Map
     sections (one per required primitive)
```
