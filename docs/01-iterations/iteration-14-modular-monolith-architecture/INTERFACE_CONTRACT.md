# Iteration 14 Interface Contract

Status: implemented and contract-tested
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: internal module, configuration, architecture-check, and quality-command contracts introduced by iteration 14.

## Public Product Interfaces

Iteration 14 introduces no intentional public HTTP, JSON, SSE, route, or
persisted product-schema change. Frozen iteration-13 black-box behavior is an
input contract to this refactor. Any discovered need for a public change must
stop the affected wave and update this contract, client/server types, tests,
and architecture impact classification before implementation.

## Backend Module Contract

Each primary capability exposes one root facade package. Consumers may use
only exported facade operations and exported immutable request/result types.
All implementation packages live below the capability's nested `internal`
directory.

Conceptual target:

```text
internal/modules/<capability>/
  api.go
  types.go
  internal/
    domain/
    application/
    ports/
    adapters/
```

Directories appear only when the capability needs that responsibility. Module
facades use business verbs, not generic file verbs. They do not expose physical
paths, untyped JSON maps, concrete adapters, or mutable internal entities.

## Cross-Module Contract

A consuming module defines the smallest port it needs. `app/integration`
implements that port by calling the provider module's public facade and mapping
bounded immutable values. Integration code contains no authorization,
state-transition, persistence, or retry decision.

Initial protected boundaries include:

```text
TeacherStreamProvider
AssistantExecutor
TaskStore
AssetCommitter
ProjectReader
LegacyReader
AtomicFileSystem
EventPublisher
Clock
IDGenerator
```

Names may become capability-specific during planning. Their responsibilities
and dependency direction may not be weakened without updating this contract.

## Error Contract

Module facades classify errors as:

| Class | Meaning | HTTP default |
|---|---|---:|
| `invalid-input` | malformed or unsafe caller input | 400 |
| `not-found` | requested owned resource is absent | 404 |
| `conflict` | stale identity or invalid state transition | 409 |
| `unavailable` | provider, executor, or bounded capacity unavailable | 503 |
| `corrupt-data` | canonical or compatibility data cannot be safely decoded | 500 with recovery guidance |
| `internal` | unclassified implementation failure | 500 |

Adapters retain wrapped technical causes for logs and tests. Transport maps the
classification without exposing secret values or unrestricted paths.

## Configuration Contract

The canonical logical shape contains these top-level sections:

```text
server
workspace
ai
assistant
image
ui
```

Precedence is:

```text
compiled defaults < config.local.json < LLL_* environment < explicit CLI flags
```

The checked-in JSON schema and example document fields, types, enums, defaults,
and secret-handling guidance. The loader accepts the existing flat keys during
the compatibility period, normalizes in memory, and does not write during
load. Conflicting old and new keys resolve in favor of the new sectioned key
and emit a non-secret warning.

## Architecture Check Contract

`tools/archcheck` exits non-zero and emits a human-readable violation plus a
machine-readable report when an import violates the approved graph. At minimum
it checks transport-to-private, cross-module-private, platform-to-business, and
frontend cross-feature-internal dependencies.

## Quality Command Contract

| Command | Required contents |
|---|---|
| `npm run check:fast` | formatting/static checks, architecture check, focused unit suites |
| `npm run check` | fast gate plus complete Go/frontend tests, contracts, and production builds |
| `npm run check:full` | complete gate plus critical browser paths and Windows-native smoke checks |

Commands return zero only when every included check passes. Provider credentials
are not required for ordinary CI; credentialed native-provider validation
remains an explicitly recorded local delivery check.

## Frontend Feature Contract

Each feature exposes a single `index.ts`. Another feature may import that file
but not its components, hooks, API adapters, tests, or state implementation.
`app` owns router/provider composition and the one global SSE subscription.
`shared` contains only behavior with multiple real feature consumers and may
not depend on a feature.
