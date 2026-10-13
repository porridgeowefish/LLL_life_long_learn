# Reasoning Primitives

This directory holds reasoning primitives used by retained compatibility-agent code. Current API teaching resources live in learning-prompts/. Each `.md` file is one primitive, loaded by `backend-go/internal/modules/assistant/internal/prompt` at prompt-build time and inlined into the prompt sent to the native CLI.

See `docs/harness-skill/facts/prompts.md` for current teaching-resource and compatibility boundaries. Registry declarations and the primitive loader own compatibility selection and expansion.

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

### Intro Agent (required)

- `knowledge_anchor.md` — Activate prior knowledge
- `boundary_map.md` — shared with Explain

### Practice Agent (required)

- `transfer.md` — Apply learned concept to a new problem

## Conventions

- File name = primitive id used in `learning-agents/registry/*.json`.
- Fixed sections per file (see `AGENT_PRIMITIVES.md`).
- Roughly 400 words per file to keep the assembled prompt manageable.
- No behavioral commands — primitives describe mechanisms, charters give orders.
