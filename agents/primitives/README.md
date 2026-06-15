# Reasoning Primitives

This directory holds the shared library of thinking mechanisms used by LLL agents. Each `.md` file is one primitive, loaded by `backend-go/internal/promptassembly` at prompt-build time and inlined into the prompt sent to Claude.

See `docs/00-product-and-architecture/AGENT_PRIMITIVES.md` for the design rationale, the mechanism-to-agent ownership table, and the file-shape contract.

## Index

### Explain Agent (required)

- `research_question_frame.md` — Tutorial-wide research question structure
- `prerequisite_scaffold.md` — Minimal prerequisite teaching without learner diagnosis
- `misconception.md` — Three-plus common misconceptions with counterexamples
- `boundary_map.md` — Scope boundary used once in the overview (shared with Intro)

### Explain Agent (optional)

- `analogy.md` — Engineering / CS analogy mapping
- `mece_decompose.md` — Optional decomposition when one stable axis exists
- `concept_graph.md` — Optional graph for non-linear concept relations
- `first_principles.md` — Final core-viewpoint compression only; never a standalone section

### Intro Agent (required, primitive ready — agent pending)

- `knowledge_anchor.md` — Activate prior knowledge
- `boundary_map.md` — shared with Explain

### Practice Agent (required, primitive ready — agent pending)

- `transfer.md` — Apply learned concept to a new problem

### Extend Agent (required, primitive ready — agent pending)

- `critical_thinking.md` — Counterfactual + relation reasoning

### Summary Agent (required, primitive ready — agent pending)

- `review_pack.md` — Feynman checklist + spaced repetition cards

## Conventions

- File name = primitive id used in `agents/registry/*.json`.
- Five fixed sections per file (see `AGENT_PRIMITIVES.md`).
- Roughly 400 words per file to keep the assembled prompt manageable.
- No behavioral commands — primitives describe mechanisms, charters give orders.
