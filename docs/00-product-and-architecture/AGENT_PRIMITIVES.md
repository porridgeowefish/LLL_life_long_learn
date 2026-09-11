# Agent Primitives

Status: active compatibility contract
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: file-shape rules for the reusable reasoning-primitives library and retained compatibility-agent coverage.

> Current-context boundary: the teacher uses a unified system prompt. This
> primitive table applies to retained Intro/Explain/Practice compatibility
> charters; it does not define active product navigation.

## Intent

LLL agents share a common library of **reasoning primitives**: reusable thinking mechanisms (MECE decomposition, first-principles reasoning, misconception hunting, etc.) loaded into the prompt at assembly time.

```text
one primitive = one mechanism = one .md file under learning-agents/primitives/
each agent charter declares which primitives it requires / optionally uses
promptassembly expands primitives into the prompt before Claude is launched
```

This is the systematic replacement for the legacy `deepthink` skill. The goal is to:

```text
preserve every useful mechanism from deepthink
distribute ownership across retained compatibility agents
keep agents independent (each charter is self-contained)
avoid recreating a single "god skill" that does everything
```

## Design Decisions

### Why not a Claude Code skill?

```text
Claude Code skills live under ~/.claude/skills/ — outside this repo.
They cannot be versioned with the codebase.
They cannot be linted or unit-tested.
They leak across projects and across users.
```

Primitives live inside the repo (`learning-agents/primitives/*.md`), so they:

```text
travel with the code (git tracked)
can be validated by Go tests (schema, presence, coverage)
can be reviewed in pull requests
can evolve with the rest of the system
```

### Why not embed everything in the charter?

```text
Embedding forces every future agent to either copy the mechanism
or miss it. That violates "each agent independent" only if the
copy drifts; embedding without sharing also violates "DRY" once
the second agent arrives.
```

The middle ground is **primitives declared by the charter, expanded by the runtime**:

```text
charter says: I require research_question_frame + prerequisite_scaffold + ...
runtime loads each named primitive .md and inlines it
charter stays small and self-describing
mechanism text lives in one place
```

## Mechanism Ownership Table

This table is the authoritative mapping from the legacy `deepthink` mechanism list to its new owner. The `primitives_coverage_test.go` test in the promptassembly package asserts every row has a home.

```text
┌────┬──────────────────────────┬─────────────────────────┬──────────────┐
│ #  │ Mechanism                │ Primitive file          │ Owner agent  │
├────┼──────────────────────────┼─────────────────────────┼──────────────┤
│  1 │ Research question frame  │ research_question_frame.md│ Explain    │
│  2 │ Prerequisite scaffold    │ prerequisite_scaffold.md│ Explain+Intro│
│  3 │ MECE decomposition       │ mece_decompose.md       │ Explain opt. │
│  4 │ First-principles         │ first_principles.md     │ Explain opt. │
│  5 │ Concept graph (Mermaid)  │ concept_graph.md        │ Explain opt. │
│  6 │ Misconception hunting    │ misconception.md        │ Explain      │
│  7 │ Boundary mapping         │ boundary_map.md         │ Explain+Intro│
│  8 │ Analogy (CS / engineering│ analogy.md              │ Explain opt. │
│  9 │ Knowledge anchor         │ knowledge_anchor.md     │ Intro        │
│ 10 │ Knowledge transfer       │ transfer.md             │ Practice     │
└────┴──────────────────────────┴─────────────────────────┴──────────────┘
```

`Progressive learning path (Step 1-4)` is not in this table — it described a
retired staged workflow rather than one reusable mechanism.

## Primitive File Shape

Every primitive `.md` follows this fixed five-section shape, kept under roughly 400 words so the assembled prompt stays manageable.

```markdown
# <Primitive Name>

## Mechanism
What this primitive is, what cognitive problem it solves, theoretical basis (1-2 sentences).

## When To Use
Trigger conditions, applicable scenarios, scenarios where it must NOT be used.

## Output Form
Required section heading, length floor/ceiling, mandatory sub-elements, format (Mermaid / list / prose).

## Anti-patterns
Three common misuse patterns to refuse.

## Example
One complete example for reference (not for copying).
```

## How To Add A New Primitive

```text
1. Write learning-agents/primitives/<name>.md following the five-section shape.
2. Decide ownership: which agent's user story needs this mechanism.
3. Add the primitive id to that agent's registry JSON under primitives.required or primitives.optional.
4. Add or update the corresponding row in the Mechanism Ownership Table above.
5. Run go test ./backend-go/... — charter_self_contained_test will refuse to compile if the file is missing or the agent declares more than six required primitives.
```

## How To Declare A Primitive In A Charter

```markdown
## Required Primitives
- `mece_decompose` — what this primitive must produce in this charter
- `first_principles` — ...

## Optional Primitives
- `analogy` — activation rule (e.g. "when intent mentions a CS / engineering background")
```

The runtime reads these declarations, loads the named files, and inserts a `# Reasoning Primitives` block into the prompt after the `# Charter` block.

Required primitive bodies define tutorial-wide obligations. Optional primitive
bodies are injected only as reference material: their presence in the prompt
does not require a matching page or section. The charter's activation
conditions decide whether an optional mechanism is used.

## Anti-patterns

```text
DO NOT put behavioral commands inside a primitive.
   Primitives describe mechanisms, charters give orders.

DO NOT declare more than six required primitives per agent.
   That recreates a single "god skill" and defeats the design.

DO NOT let two primitives own the same output section.
   Each Output Contract section maps to at most one primitive.

DO NOT let optional Explain primitives become a repeated page template.
   MECE and concept graphs are used only when the topic structure benefits.
   First-principles reasoning is limited to final core-viewpoint compression
   and never owns a standalone section.

DO NOT inline a primitive's text into a charter by hand.
   Always reference by id so the test suite can verify coverage.
```

## Verification

```text
go test ./backend-go/internal/agentregistry/...
  → charter_self_contained_test asserts every declared primitive file exists
    and every agent has a non-empty userStory

go test ./backend-go/internal/promptassembly/...
  → primitives_loader_test asserts load + cache behavior
  → assembly_primitives_test asserts the expanded prompt contains
    the # Reasoning Primitives block and the required primitive bodies

go test ./backend-go/... -run TestPrimitivesCoverage
  → primitives_coverage_test asserts every legacy mechanism in the
    table above resolves to an existing primitive file
```

## Related Documents

```text
docs/00-product-and-architecture/LEARNING_PROJECT_STRUCTURE.md
  Legacy compatibility model and the agent invocation contract.

docs/00-product-and-architecture/AGENT_ARCHITECTURE.md
  Execution principle: agents coordinate via files, not message buses.

docs/00-product-and-architecture/API_CONTRACT_STRATEGY.md
  HTTP surface for agent invocation.

learning-agents/charters/explain.md
  First production charter, demonstrating the User Story + Required
  Primitives + Optional Primitives + Output Contract + Out of Scope shape.

learning-agents/registry/explain.json
  First production registry entry, demonstrating the
  userStory + primitives.{required,optional} schema.
```

## Status

```text
Wave 1 — skeleton in place ✓
Wave 2 — six Explain primitives written (5 required + 1 optional) ✓
Wave 3 — compatibility-agent primitives written ✓
Wave 4 — registry schema + explain charter migration ✓
Wave 5 — promptassembly loader + assembly block ✓
Wave 6 — test suite green (registry, promptassembly, coverage) ✓
Wave 7 — full version of this document + ROADMAP.md ✓
```

`go test ./backend-go/...` is the regression gate. If a primitive file goes missing or a charter declares an invalid primitive, the test suite refuses to compile.
