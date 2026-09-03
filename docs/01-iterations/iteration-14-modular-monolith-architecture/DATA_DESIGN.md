# Iteration 14 Data Design

Status: approved design; implementation not started
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: configuration, quality-artifact, fixture, and compatibility data boundaries for iteration 14.

## Product Data

Iteration 14 does not change canonical learner project paths or schemas.
Iteration-13 conversation, task, asset, source, migration, project, and global
preference files remain byte-compatible inputs and outputs. Source moves in the
repository do not authorize moving files below `projects/`.

## Write Ownership

| Data family | Canonical writer | Other-module access |
|---|---|---|
| project metadata, classification, scope | projects/learning facade | immutable projection |
| conversation and teacher recovery | teacher facade | immutable context or summary |
| assistant tasks and attempts | assistant facade | immutable task projection |
| assets, versions, annotations | assets facade | immutable asset projection |
| sources and revisions | sources facade | immutable source projection |
| global preferences | preferences facade | bounded read-only snapshot |

Filesystem platform primitives do not know these business paths. Physical path
and serialization rules remain inside the owning module adapter.

## Local Configuration

`config.local.json` remains local and ignored. The target sectioned form is:

```json
{
  "server": {"host": "127.0.0.1", "port": 8787},
  "workspace": {"root": "."},
  "ai": {
    "defaultProvider": "primary",
    "searchEngine": "google",
    "providers": [
      {
        "id": "primary",
        "kind": "openai",
        "name": "Primary provider",
        "baseURL": "",
        "apiKeyEnv": "LLL_PRIMARY_AI_API_KEY",
        "model": "",
        "contextWindowTokens": 0,
        "reasoning": false,
        "thinking": false
      }
    ],
    "bindings": {
      "teacher": {"providerId": "primary", "model": ""},
      "conversationCompaction": {"providerId": "primary", "model": ""},
      "annotationAskAI": {"providerId": "primary", "model": ""}
    }
  },
  "assistant": {
    "runtime": "claude",
    "maxConcurrent": 5,
    "maxConcurrentPerProject": 2,
    "bins": {}
  },
  "image": {
    "enabled": false,
    "baseURL": "",
    "apiKeyEnv": "LLL_IMAGE_API_KEY",
    "model": "",
    "promptModel": "",
    "pythonBin": "python",
    "backup": {
      "baseURL": "",
      "apiKeyEnv": "LLL_IMAGE_BACKUP_API_KEY",
      "model": ""
    }
  },
  "ui": {"theme": "lychee-paper"}
}
```

These are the canonical iteration-14 names. Existing keys
`askAiProviders`, `agentRuntime`, `agentRuntimeBins`, `imageApiKey`,
`imageBaseURL`, `imageModel`, `imageBackupApiKey`, `imageBackupBaseURL`,
`imageBackupModel`, `imagePromptModel`, `pythonBin`, and `ui.theme` remain
compatibility inputs. Existing inline provider `apiKey` values remain readable
during the compatibility period and remain writable through the existing local
Settings flow. When both are present, the environment value named by
`apiKeyEnv` wins. The committed example omits inline `apiKey` and contains no
secret. `LLL_AGENT_RUNTIME` and runtime-specific binary environment variables
remain supported. `WORKSPACE` remains a compatibility environment input; the
canonical override is `LLL_WORKSPACE_ROOT`.

The typed in-memory value carries provenance per resolved section for
diagnostics, but provenance is not written back to the local file.

## Quality Artifacts

Generated evidence lives below:

```text
.artifacts/quality/
  tests/
    go-test.json
    frontend-junit.xml
    e2e-junit.xml
  coverage/
    go.out
    go.html
    frontend-lcov.info
    frontend-html/
  architecture/
    dependency-report.json
```

`.artifacts/` is ignored. CI may upload the directory as an ephemeral build
artifact. Reports must not contain API keys, complete prompt content, learner
preferences, uploaded source contents, or unrestricted absolute paths.

## Fixtures

Cross-module fixtures live below `tests/fixtures/` and contain only synthetic
or anonymized data:

```text
canonical/iteration-13/
legacy/zones/
legacy/memory-era/
corrupt/
```

Module-local storage examples use Go `testdata/` directories or frontend test
fixtures beside their owner. A fixture documents its originating schema and
expected read projection. Tests never mutate the checked-in fixture directly;
they copy it to a temporary workspace.

## Migration And Rollback

The source refactor performs no product-data migration. Old configuration is
normalized only in memory. Existing iteration-13 migration journals and
backups retain their original meaning.

Because product schemas remain unchanged, rollback uses an earlier binary or
Git commit without converting learner data. Any implementation discovery that
requires a persisted change triggers a separate data-contract amendment,
migration design, and approval before that wave proceeds.
