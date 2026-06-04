# Data Model

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: conceptual persistence model for upcoming iterations.

## Initial Entities

```text
study_topics
- id
- title
- goal
- learner_level
- domain_context

task_runs
- id
- topic_id
- title
- prompt
- status
- command
- cwd
- created_at
- started_at
- finished_at

task_logs
- id
- task_run_id
- stream
- text
- created_at

study_artifacts
- id
- task_run_id
- artifact_type
- content
- created_at
```

## Current Reality

The current implementation is in-memory only. Persistent storage is a future slice, so this file is a planning model, not yet a runtime contract.
