# ADR-0019 — Vision OCR, Web Search Tool, And Repository Normalization

Status: accepted
Date: 2026-09-11

## Decision

**OCR as a vision-model call.** Image sources (png/jpg/jpeg/webp/gif/bmp/tiff)
uploaded with parse approval route to a dedicated OCR step when
`askAiProviders.bindings.ocr` resolves to a vision-capable provider; the step
extracts verbatim text (structure preserved, formulas as LaTeX) into the same
single `derived/content.md` the teacher citation path already reads, then
flips the source to `ready`. Without the binding, images keep the iteration-16
CLI-agent parse path — OCR is an explicit opt-in, not a silent default. The
existing cloud-disclosure gate at upload is the privacy boundary; failures
mark the source `failed/ocr-failed` and keep the original.

**Web search as a bounded teacher tool.** With a `webSearch` config section
(provider `zhipu`, engine, key or env key), the teacher registers a second
tool `search_web` alongside `delegate_learning_work`. The service executes
searches itself through a small `websearch.Searcher` interface (Zhipu
web-search API behind it) and feeds results back to the provider via a tool
loop capped at two searches per turn. Search needs no learner approval
(read-only public web); the system prompt requires source links and
reliability caveats. Unconfigured leaves the tool unregistered and teacher
behavior unchanged. On the anthropic provider kind, thinking is disabled for
turns that register tools, because re-synthesized assistant messages cannot
replay signed thinking blocks; the openai kind is unaffected.

Gateway messages gained `ToolCalls` / `ToolCallID` fields mapped to the
openai `tool_calls`/`role=tool` shape and the anthropic
`tool_use`/`tool_result` blocks; the delegate tool remains one-shot.

**Repository normalization.** `agents/` is renamed `learning-agents/` (one
path constant plus registry JSON charter paths; HTTP paths unchanged) so
learner-facing learning agents stop colliding with the repo-maintenance
"coding agents" concept in AGENTS.md. Dev check scripts move to
`tools/check/`; manual QA scripts move to `tests/manual/` with a README.
The runtime folder ledger relocates from the workspace root to
`projects/folders.json` with a one-time legacy read-back copy on first open
(legacy file untouched).

## Consequences

- OCR quality equals the configured vision model; no local OCR runtime is
  shipped (deliberate: Windows packaging cost outweighed the benefit).
- Search provider swap is local to one package; the loop cap bounds latency
  and cost per turn.
- Anthropic-kind teachers trade thinking summaries for tool availability on
  search-enabled turns; revisit if raw block replay lands.
- Repository moves are `git mv` history-preserving; the folder-store
  migration is copy-forward, never destructive.

## Supersedes

Extends ADR-0015's modular monolith layout section and ADR-0017's source
contract; no decisions are reversed.
