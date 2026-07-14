# Agent Rules Index

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: this directory owns detailed atomic AI agent rules for LLL.

## Design Goal

Use progressive disclosure:

```text
AGENTS.md keeps the permanent highest-priority behavior.
This README maps tasks to rule files.
Atomic rule files hold detailed execution guidance.
```

## Atomic Rule Files

- [00-startup-and-source-of-truth.md](./00-startup-and-source-of-truth.md)
- [10-documentation-governance.md](./10-documentation-governance.md)
- [30-api-contracts.md](./30-api-contracts.md)
- [40-testing-and-verification.md](./40-testing-and-verification.md)
- [60-task-orchestration-safety.md](./60-task-orchestration-safety.md)
- [90-simplicity-and-refresh.md](./90-simplicity-and-refresh.md)

## Trigger Map

```text
New task start:
00-startup-and-source-of-truth.md

Docs update or new project rule:
00-startup-and-source-of-truth.md
10-documentation-governance.md

New iteration, iteration renumbering, product/domain change, or ADR:
00-startup-and-source-of-truth.md
10-documentation-governance.md
40-testing-and-verification.md

API or frontend/backend wiring:
00-startup-and-source-of-truth.md
30-api-contracts.md

Validation and delivery:
40-testing-and-verification.md
90-simplicity-and-refresh.md

Claude Code process execution, streaming, and cancellation:
60-task-orchestration-safety.md
```
