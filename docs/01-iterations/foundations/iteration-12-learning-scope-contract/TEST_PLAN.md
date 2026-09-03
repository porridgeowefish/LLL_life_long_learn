# Test Plan

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28

## Automated Coverage

- learning-scope draft creation and map snapshot copying;
- duplicate sibling concept ownership rejection;
- discipline catalog endpoint;
- unknown map-topic rejection;
- snapshot stability after map regeneration;
- encyclopedia prompt declares both output files;
- zone prompt embeds scope and Intro non-expansion rules;
- artifact watcher recognizes both new root artifacts;
- frontend sends topic ID through overview, task plan, and creation modal;
- optional learning-goal fields and templates remain covered.

## Commands

```text
cd backend-go
go test ./internal/learningscope ./internal/workspace ./internal/artifactwatch ./internal/promptassembly ./internal/server

cd frontend
npm run test -- --run src/components/feature/project/CreateProjectModal.test.tsx src/components/feature/project/DisciplineOverview.test.tsx src/hooks/useArtifactRefresh.test.tsx
npm run build
```

## Residual Risk

The contract strongly guides generated content and validates structural
ownership, but it does not semantically inspect every generated tutorial
paragraph. A future evaluator may compare manifests or pages to scope concepts
without changing this persisted contract.
