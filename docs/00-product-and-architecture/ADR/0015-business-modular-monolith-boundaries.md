# ADR-0015: Business-Capability Modular Monolith Boundaries

Status: accepted and implemented in iteration 14
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: durable decision for LLL source-code module boundaries, dependency direction, configuration ownership, and architecture enforcement.

## Context

LLL remains a Windows-first, local-first application delivered as one Go
process plus one React SPA. The filesystem under `projects/` is canonical
product data. There is no database-first truth, external broker, cloud account,
or microservice deployment requirement.

The implementation has nevertheless grown across roughly thirty-five flat Go
packages. `internal/server` can import almost every store and runtime package,
and several files combine application rules with transport, filesystem, or
Windows process concerns. Frontend behavior is similarly split across global
`api`, `pages`, `components`, `hooks`, and `store` directories even when those
files change as one product capability.

The result is avoidable coupling: a maintainer must reconstruct one use case
from technical folders, route handlers can bypass application services, and
multiple packages can learn the same persisted path or runtime detail.

## Decision

LLL will become a business-capability modular monolith. It remains one
deployable application, while source code is organized around these primary
capabilities:

```text
teacher
assistant
assets
sources
projects and learning
preferences
legacy compatibility
```

Each capability exposes one small public facade. Its domain rules,
application operations, ports, file adapters, and tests remain private behind
that facade. Go nested `internal` packages and automated dependency checks
enforce the boundary.

Complex modules may use the four conceptual responsibilities
`domain/application/ports/adapters`. These are responsibilities, not a
mandatory empty-directory template. A simple module stays compact until its
rules, failure modes, or external boundaries justify further separation.

## Target Dependency Direction

```text
cmd -> app/bootstrap
app/bootstrap -> transport + module facades + platform implementations
transport -> module facades
module application -> module domain + module-owned ports
module adapters -> module-owned ports + platform primitives
platform -> standard library and non-business technical dependencies
```

Only `app/bootstrap` may assemble concrete implementations. Cross-module glue
lives in `app/integration`, performs bounded type conversion, and contains no
business decisions. A module may not import another module's private
implementation or manipulate another module's persisted files.

## Data And Write Ownership

Canonical write authority is singular:

| Data | Owning module |
|---|---|
| project identity, classification, and learning scope | projects/learning |
| conversation events and teacher response recovery | teacher |
| assistant task, attempt, lease, and result state | assistant |
| asset content, versions, annotations, and commit journals | assets |
| source originals, revisions, parsing state, and tombstones | sources |
| `<WORKSPACE>/preferences.md` | preferences |

The platform filesystem package owns path safety, atomic primitives, and file
mechanics. It does not own business paths or schemas. Compatibility readers may
read legacy Intro, Explain, Practice, Extend, Summary, and memory-era data, but
new behavior does not write those formats.

## Selective Ports

Interfaces protect a demonstrated variation or failure boundary. Initial
required boundaries include model streaming, visible CLI execution, task
persistence, asset commit, canonical and legacy project reads, atomic file
operations, event publication, clocks, and identifiers.

LLL will not create one interface per function, generic repository bases,
duplicate DTO layers, CQRS, a domain event bus, or a workflow engine merely to
match a diagram.

## Configuration And Quality

One typed platform configuration loader owns defaults, local JSON, environment
overrides, validation, and redaction. Modules receive only their typed config
slice. Existing flat local keys remain read-compatible during migration and
are never rewritten automatically.

Architecture boundaries are executable contracts. Repository quality commands
cover formatting, static analysis, dependency checks, unit and integration
tests, API and file contracts, builds, browser paths, coverage reports, and
Windows-native CLI smoke checks. Generated evidence is ignored locally and
uploaded as CI artifacts.

## Migration

Iteration 14 performs a behavior-preserving, multi-wave source migration on one
branch and merges once after the complete gate passes. Public APIs and product
file schemas are frozen during the move. Existing data is neither deleted nor
rewritten. Temporary forwarding facades may exist within the branch, but the
final composition root may not retain old bypass paths.

## Alternatives Considered

### Global technical layers

Global `handlers/services/repositories/models` folders reduce naming
inconsistency but continue to scatter one business change across the tree.

### Strict repository-wide hexagonal architecture

Global `domain/application/ports/adapters` layers provide stronger uniformity,
but impose excess interfaces, DTO mapping, and navigation on stable single-
implementation behavior. LLL adopts their dependency direction selectively
inside complex capability modules.

### Microservices

Independent deployment would add network failure, distributed consistency,
service discovery, and operational cost without solving a current product
requirement. It is rejected.

## Consequences

Positive consequences:

- one product capability becomes one primary reading and change location;
- transport and platform details cannot bypass module rules;
- persistence ownership and compatibility behavior become explicit;
- new iterations gain repeatable code, test, config, and documentation homes;
- future extraction remains possible if a real independent deployment need
  appears.

Costs:

- iteration 14 is a large source move with substantial import churn;
- public module facades and architecture tests require ongoing discipline;
- some integration adapters are necessary to prevent direct module coupling;
- Windows-native behavior still requires real local smoke validation beyond
  ordinary unit tests.
