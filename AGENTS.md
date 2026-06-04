# AGENTS.md: LLL Agent Entry

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: lightweight entry point for all AI coding agents.

## 0. Working Rule

Keep this file short. Read this file first, then:

```text
docs/INDEX.md
docs/00-product-and-architecture/agent-rules/README.md
```

Read only the rule files triggered by the current task. Do not preload the whole docs tree.

## 1. Team Principles

All agents must follow:

```text
以瞎猜接口为耻，以认真查询为荣。
以模糊执行为耻，以寻求确认为荣。
以跳过验证为耻，以主动测试为荣。
以平行造轮子为耻，以复用现有为荣。
以假装理解为耻，以诚实说明为荣。
以脱离文档为耻，以维护事实源为荣。
以偷改范围为耻，以遵守迭代边界为荣。
以只改代码不改文档为耻，以同轮同步为荣。
```

Execution meaning:

```text
Before changing behavior, check code, API, docs, and current iteration scope.
If a rule or fact is missing, add or update the right document in the same task.
Prefer incremental structure over speculative abstraction.
Explain validation status and residual risk when you cannot fully verify.
```

## 2. Progressive Disclosure

Use:

```text
docs/00-product-and-architecture/agent-rules/README.md
```

That index decides which atomic rules to read for:

```text
startup
documentation governance
API contracts
testing
task orchestration and safety
long-task refresh
```
