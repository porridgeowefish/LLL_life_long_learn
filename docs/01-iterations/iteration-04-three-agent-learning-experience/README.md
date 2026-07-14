# Iteration 04: Three-Agent Learning Experience

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: this directory defines the fourth LLL delivery slice.

> Partially superseded by iterations 07 and 09: prerequisite diagnosis is
> retained, while ADR-0008 replaces both child projects and `快速补充` with
> generated gap summaries inside the current page.

## Goal

Refactor the three primary learning agents without changing the five-zone architecture:

```text
Intro    -> 探索：校准背景、具体样例与前置缺口诊断
Explain  -> 学习：研究问题框架、必要前置、manifest 多页与追问页
Practice -> 出题：一至五星、客观到主观、即时判题与项目成长值
```

Extend and Summary remain unchanged. Agent IDs and zone IDs remain stable.

## Included

```text
intro/survey.json + intro/output.md + intro/assessment.json
confirmed creation of prerequisite child projects (historical implementation; superseded in iteration 07)
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

- Historical decision: prerequisite child projects required learner confirmation. Iteration 07 replaces this default with an inline prerequisite bridge.
- Explain pages are manifest-owned; a follow-up appends a page and never overwrites its parent.
- Answer keys are readable by the backend but forbidden through `/files`.
- Growth uses idempotent append-only events with policy `practice-v1`.
- Deferred iter-03 proposal work moves to iter-05 or later.
