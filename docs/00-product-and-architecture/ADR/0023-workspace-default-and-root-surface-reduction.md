# ADR-0023 — Workspace Default And Root Surface Reduction

Status: accepted
Date: 2026-09-24

## Decision

**Learner workspace defaults to `<repository>/workspace/`.** The compiled
default for the workspace root (package `filesystem` init and the config
loader's file-lookup base) changes from the repository root itself to
`<repo>/workspace/`, and both derivations converge on one function,
`filesystem.DefaultWorkspaceRoot`, so the config-file lookup and the runtime
path variables can never disagree again. All learner-owned runtime data now
lives under that single gitignored directory by default: `projects/`
(including the folders ledger), `preferences.md`, `config.local.json`,
workspace-local overrides, and `tmp/` diagnostics (teacher stream-shape log,
desktop-launcher PID and logs). Existing override channels are unchanged:
`--workspace`, `LLL_WORKSPACE_ROOT` / `WORKSPACE`, and `workspace.root` in
`config.local.json`. `ConfigureWorkspace` creates the directory eagerly so
every launch mode (npm start, go run, desktop shortcut) gets the same layout.

**learning-agents/ is product content with a shipped-default fallback.** The
registry resolves the agents root as: test override, then
`<workspace>/learning-agents/` when that directory exists, then the
repository's tracked `learning-agents/`. A custom workspace can still ship a
complete replacement agent set; a fresh workspace no longer needs a copy of
tracked content inside itself.

**gen_infographic.py is backend-invoked product runtime, not manual QA.**
ADR-0019 moved it to `tests/manual/` while the explain infographic pipeline
still resolved `WORKSPACE/scripts/gen_infographic.py`, silently breaking
image generation. The script returns to `scripts/` (git mv, history
preserved): `scripts/` holds operator entry points plus backend-invoked
runtime scripts, and a workspace-local `scripts/gen_infographic.py` still
overrides the shipped one.

**Local-only evidence consolidates.** Root `tmp/` debugging residue moves to
`.artifacts/` (incident forensics) or regenerates at `<workspace>/tmp/`
(diagnostics). The duplicated legacy-shape `config.local.example.json` at the
repository root is removed; `config/config.example.json` is the single
committed template.

## Consequences

- The repository root keeps only product code, contracts, and tooling;
  runtime data, diagnostics, and build or quality output are gitignored
  directories (`workspace/`, `.artifacts/`, `dist/`).
- A one-time operator migration moved the existing root-level `projects/`,
  `preferences.md`, `config.local.json`, and the legacy `folders.json` into
  `workspace/` (copy-forward semantics; the legacy `folders.json` stays inert
  because the canonical ledger already lives at `projects/folders.json`).
- The preferences recovery backup re-keys itself to the new workspace path on
  the first save; the moved `preferences.md` itself carries the content.
- Tests pin the three behaviors: the converged default workspace base, the
  registry shipped-set fallback, and the infographic script resolution order.

## Supersedes

Amends ADR-0019's repository-normalization section (script taxonomy and
agents-root resolution); no earlier decision is reversed.
