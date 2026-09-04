# LLL Configuration

Status: active
Owner: project maintainer
Last reviewed: 2026-09-04
Source of truth: `config.example.json` (shape) and `backend-go/internal/platform/config` (loader behavior). The iteration-14 data contract is `docs/01-iterations/iteration-14-modular-monolith-architecture/DATA_DESIGN.md`.

## File

`config.local.json` in the workspace root is the local configuration file. It
is gitignored and is never rewritten during load.

## Precedence

```text
compiled defaults < config.local.json < LLL_* environment < explicit CLI flags
```

Key environment variables:

```text
LLL_SERVER_PORT          server.port
LLL_WORKSPACE_ROOT       workspace.root (canonical)
WORKSPACE                workspace.root (compatibility alias)
LLL_AGENT_RUNTIME        assistant.runtime
LLL_PRIMARY_AI_API_KEY   ai provider key when apiKeyEnv names it
LLL_IMAGE_API_KEY        image primary key
LLL_IMAGE_BACKUP_API_KEY image backup key
CLAUDE_BIN / CODEBUDDY_BIN / HERMES_BIN / CODEX_BIN / TRAE_BIN   runtime binaries
```

CLI flags (added in iteration 14): `--port`, `--workspace`, `--config`.

## Legacy flat keys (read-compatible)

`askAiProviders`, `agentRuntime`, `agentRuntimeBins`, `imageApiKey`,
`imageBaseURL`, `imageModel`, `imageBackupApiKey`, `imageBackupBaseURL`,
`imageBackupModel`, `imagePromptModel`, `pythonBin` remain readable. When
both a sectioned key and its legacy alias exist, the sectioned key wins and
loading emits a non-secret warning. Existing inline `apiKey` values stay
readable; the environment variable named by `apiKeyEnv` wins over inline
keys. The committed example contains no secrets.
