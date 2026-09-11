# Quality Commands

Status: active
Owner: project maintainer
Last reviewed: 2026-09-04
Source of truth: repository quality-gate definitions introduced by iteration 14.

## Commands

| Command | Purpose | Contents |
|---|---|---|
| `npm run check:fast` | pre-change feedback loop | gofmt check, `go vet`, `go build ./...`, architecture check, short-mode Go tests, frontend lint, frontend tests |
| `npm run check` | pre-merge gate | everything in `check:fast` plus full Go tests with coverage, Go coverage floor vs baseline, frontend production build, backend production build |
| `npm run check:full` | release gate | everything in `check` plus the contract-freeze suite and Windows-native smoke checks (local Windows only) |

Every command returns non-zero when any included check fails. The three
levels are strictly nested: fast ⊂ check ⊂ check:full.

Individual commands remain runnable on their own:

```text
go test ./...
go run ./tools/archcheck -repo .
npm --prefix frontend run lint
npm --prefix frontend run test
npm --prefix frontend run build
```

## Evidence

Generated reports land below `.artifacts/quality/` (gitignored; CI uploads
it as an ephemeral artifact):

```text
.artifacts/quality/
  tests/            test outputs
  coverage/         go.out coverage profile + gate results
  architecture/     dependency-report.json
```

Reports must not contain API keys, prompt content, learner preferences,
uploaded source contents, or unrestricted absolute paths.

## Coverage Policy

`tests/baseline.json` records the pre-refactor statement coverage
(iteration-13 code: Go 29.35%, frontend 40.24%). `tools/check/check-coverage.js`
fails the gate when measured coverage drops more than one percentage point
below the baseline. The baseline is re-recorded deliberately — never lowered
silently to make a failing gate pass.

## CI

`.github/workflows/ci.yml` runs `npm run check` on windows-latest without
provider credentials. Credentialed native-provider validation and the real
Windows-native CLI smoke stay local delivery checks recorded in the active
iteration's `DELIVERY_NOTES.md`.

## Architecture Check

`tools/archcheck` enforces the ADR-0015 dependency graph:

```text
R1 transport must not import modules/*/internal/**
R2 a module must not import another module's internal/**
R3 platform must not import modules, transport, or app
R4 a frontend feature may not import another feature's non-index file
```

Temporary exemptions during the migration waves live in an allowlist file
passed via `-allow` and must be gone by the final cutover wave.
