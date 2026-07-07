# Iteration 04: Three-Agent Learning Experience

Status: implemented
Owner: project maintainer
Last reviewed: 2026-06-14
Source of truth: this directory defines the fourth LLL delivery slice.

## Goal

Refactor the three primary learning agents without changing the five-zone architecture:

```text
Intro    -> 探索：校准背景、具体样例、前置缺口与子项目建议
Explain  -> 学习：研究问题框架、必要前置、manifest 多页与追问页
Practice -> 出题：一至五星、客观到主观、即时判题与项目成长值
```

Extend and Summary remain unchanged. Agent IDs and zone IDs remain stable.

## Included

```text
intro/survey.json + intro/output.md + intro/assessment.json
confirmed creation of prerequisite child projects
explain/manifest.json + explain/pages/*.md
follow-up pages linked by parentPageId
practice/tasks.json schema v2 + private answer-key.json
objective per-question checking; subjective batch evaluation
progress/events.jsonl + progress/summary.json
legacy output.md and tasks.json reads
```

## Excluded

```text
global points, ranks, streaks, or achievements
automatic prerequisite project creation
per-question subjective evaluation
Extend and Summary redesign
memory/summary proposal workflows
PTY session-continuity redesign
```

## Key Decisions

- Child prerequisite projects require learner confirmation and use the existing subproject model.
- Explain pages are manifest-owned; a follow-up appends a page and never overwrites its parent.
- Answer keys are readable by the backend but forbidden through `/files`.
- Growth uses idempotent append-only events with policy `practice-v1`.
- Deferred iter-03 proposal work moves to iter-05 or later.
