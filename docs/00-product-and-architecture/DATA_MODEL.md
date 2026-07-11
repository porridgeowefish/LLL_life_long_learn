# Data Model

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: conceptual persistence model for upcoming iterations.

## Target File-First Entities

```text
workspace_projects
- id
- slug
- title
- parent_project_id
- status
- active_zone
- root_path
- created_at
- updated_at

learning_zones
- id
- project_id
- zone_name
- path
- predecessor_zone_names
- summary_protected

agent_roles
- id
- name
- charter_path
- allowed_zone_names
- default_output_targets

claude_sessions
- id
- project_id
- zone_name
- agent_role_id
- state
- run_path
- prompt_path
- created_at
- updated_at
- finished_at

session_turns
- id
- session_id
- ordinal
- turn_type
- source
- content_path
- created_at

run_records
- id
- session_id
- stdout_path
- stderr_path
- transcript_path
- result_path
- metadata_path

learning_artifacts
- id
- project_id
- zone_name
- artifact_type
- source_session_id
- source_turn_id
- target_path
- created_at
- updated_at

project_memory_snapshots
- id
- project_id
- memory_markdown_path
- memory_state_path
- updated_at

learner_memory_snapshots
- id
- profile_path
- state_path
- updated_at

learning_events (implemented file-first in `progress/events.jsonl`)
- id
- project_slug
- source_type
- source_id
- activity_delta
- growth_delta (`delta` on disk for backward compatibility)
- title
- detail
- outcome
- created_at
```

## Current Reality

The current implementation is still mostly in-memory and task-oriented.
This file defines the persistence target for the next architecture direction.

## Persistence Rule

Use:

```text
filesystem files as canonical truth
in-memory indexes as runtime acceleration only
```

That means:

```text
project markdown and json files are durable
run folders are durable
artifact files are durable
session indexes may be rebuilt from disk if needed
```
