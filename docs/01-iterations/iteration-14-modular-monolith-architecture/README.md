# Iteration 14: Business-Capability Modular Monolith Architecture

Status: implementation complete; user-run native browser acceptance pending
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: scope, architecture decisions, and delivery waves for the non-destructive modular-monolith refactor.

## Goal

Reorganize LLL around cohesive business capabilities with enforceable
dependencies, one typed configuration system, and one repeatable quality gate.
The result must be easier to extend in later iterations and easier for a new
maintainer to read without changing learner-visible behavior or persisted
project data.

## Delivered Starting-Point Migration

Iteration 13 was the starting product baseline: a Go HTTP/SSE server, React
SPA, API teacher, visible asynchronous CLI assistants, versioned assets,
sources, global preferences, and file-first recovery. The current backend has
many flat internal packages, a transport package with broad knowledge, and
large files that combine application and infrastructure concerns. The frontend
contains feature-oriented components but its API, pages, hooks, and state are
still organized primarily by technical type.

## Target Shape

LLL remains one Go process and one React application. Backend code is grouped
under business modules for teacher, assistant, assets, sources, projects,
learning, and preferences. Legacy five-zone behavior is isolated behind a
compatibility boundary. The application bootstrap is the only composition root;
HTTP/SSE is a thin inbound adapter; platform packages contain only non-business
configuration, filesystem, process, event, and identity mechanics.

Complex modules use domain, application, ports, and adapters where those
responsibilities are real. Simple modules do not create empty layers or
single-use abstractions to satisfy a template.

The frontend follows matching feature boundaries. Each feature exposes one
public entry point; app composition and genuinely shared UI or rendering
primitives remain outside features.

## Included Scope

- freeze public API, SSE, configuration, and persisted-file compatibility;
- add current and legacy fixtures plus recovery characterization tests;
- centralize typed configuration, precedence, validation, and redaction;
- add architecture dependency checks, standardized quality commands, CI, and
  generated test/coverage reports;
- create one application bootstrap and thin HTTP/SSE transport adapters;
- move backend behavior into business capability modules with private
  implementations and explicit facades;
- split oversized teacher, assistant, workspace, launcher, and route
  responsibilities without changing their product contracts;
- establish one canonical writer for each persisted data family;
- reorganize the React frontend into app, feature, and shared boundaries;
- isolate legacy readers and migration behavior from new write paths;
- update all affected current documentation and record delivery evidence.

## Excluded Scope

- product redesign or navigation changes;
- new teacher, assistant, asset, source, or project features;
- public API or SSE shape changes;
- changes to `projects/` paths or persisted schemas;
- automatic rewriting of `config.local.json`;
- removal of legacy learner data;
- database, message broker, workflow engine, or microservice introduction;
- cloud accounts, multi-user collaboration, or independent service deployment;
- provider changes unrelated to preserving the existing boundaries.

## Delivery Waves

1. Freeze contracts, fixtures, dependency graph, and coverage baseline.
2. Add centralized configuration, architecture checks, quality commands,
   reports, and CI.
3. Extract bootstrap, integration glue, and thin HTTP/SSE transport.
4. Migrate preferences, sources, and assets as lower-risk reference modules.
5. Migrate teacher conversation, provider streaming, and recovery.
6. Migrate assistant task state, dispatcher, validation, result commit, and
   Windows CLI execution.
7. Split project, learning, filesystem, and legacy compatibility ownership.
8. Reorganize frontend app/features/shared boundaries.
9. Remove temporary bypass paths, run the full gate, synchronize docs, and
   perform the single final cutover.

Each wave is implemented as one or more reviewable commits. Every commit must
compile and keep the applicable fast gate green. The branch merges only after
the complete gate, compatibility checks, and Windows smoke paths pass.

## Confirmed Decisions

- business capabilities are the primary source-code boundary;
- complex modules selectively use hexagonal responsibilities;
- nested Go `internal` packages and automated checks enforce privacy;
- module facades and consumer-owned ports are the cross-boundary contract;
- `app/integration` connects modules but owns no business rule;
- persisted files retain one canonical writer;
- platform filesystem primitives do not own business paths or schemas;
- old formats are read-compatible and never become new write targets;
- generated reports live under `.artifacts/` and are not committed;
- current coverage is measured before thresholds are fixed;
- the refactor preserves public behavior and product storage formats.

## Documentation Impact

Architecture classification: runtime component boundaries, internal interfaces,
local configuration schema, compatibility policy, and quality enforcement.

Decision record:

- `docs/00-product-and-architecture/ADR/0015-business-modular-monolith-boundaries.md`

Synchronized long-lived owners:

- `docs/INDEX.md`
- `docs/00-product-and-architecture/README.md`
- `docs/00-product-and-architecture/SYSTEM_ARCHITECTURE.md`
- `docs/00-product-and-architecture/BACKEND_ARCHITECTURE.md`
- `docs/00-product-and-architecture/REPOSITORY_MAP.md`
- `docs/00-product-and-architecture/API_CONTRACT_STRATEGY.md`
- `docs/00-product-and-architecture/DATA_MODEL.md`
- `docs/00-product-and-architecture/ROADMAP.md`
- `docs/00-product-and-architecture/ADR/README.md`
- `docs/01-iterations/README.md`

No PRD, domain-object, learning-project storage, or Agent-role contract changes
are introduced. Those owners therefore remain unchanged.

## Completion Definition

Iteration 14 is complete only when:

```text
transport cannot import module-private stores or runtime implementations
every primary capability has one documented public facade
platform packages contain no business dependency
canonical data families retain one writer
current and legacy project fixtures remain readable
public API, SSE, and persisted schemas remain compatible
old and new configuration shapes resolve to equivalent typed values
Go, frontend, contract, build, browser, and architecture checks pass
Windows native-CLI smoke paths pass on a real Windows environment
delivery notes contain commands, results, residual risks, and migration evidence
all required long-lived documents match delivered reality
```
