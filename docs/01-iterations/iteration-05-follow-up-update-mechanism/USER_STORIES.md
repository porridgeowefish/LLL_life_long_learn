# Iteration 05 User Stories

Status: active
Last reviewed: 2026-07-05

## Explain Follow-Up

- As a learner, I want the system to make questioning feel expected, so I do
  not treat the first AI answer as the final authority.
- As a learner, I want to ask repeated questions directly in the terminal and
  have the existing page improve afterward, so I do not have to assemble
  understanding from scattered follow-up pages.
- As a learner, I want a new page only when my question opens an independent
  knowledge module, so the table of contents stays meaningful.
- As a learner, I want my own hypothesis preserved before the AI corrects or
  extends it, so I can see my thinking evolve rather than disappear.
- As a learner, I want the AI to distinguish "your idea is useful but limited"
  from "this is factually wrong", so independent thinking feels safe.
- As a learner, I want examples, counterexamples, and caveats from a follow-up
  to land near the original concept they clarify.
- As a learner, I want my question to be allowed to change the whole
  explanation structure when it exposes a better organizing frame.
- As a learner, I want to view that a page was revised and still recover the
  previous version if the update was worse.

## Navigation

- As a learner, I want the Explain navigation to avoid noisy one-off follow-up
  entries.
- As a learner, I want truly new modules to appear as first-class pages with a
  clear title, order, and relationship to the source page.
- As a learner, I want page titles and order to be revised when the old sequence
  is no longer the clearest learning path.
- As an existing user, I want old `kind=followup` pages to keep rendering.

## Frontend Boundary

- As a learner, I do not want another frontend question box for follow-up; the
  terminal is the interaction surface.
- As a learner, I want the frontend to stay focused on rendering the updated
  artifact and navigation, not duplicating the Agent conversation.

## Agent Behavior

- As a learner, I want the agent to answer doubts as part of the study artifact,
  not as a detached chat reply.
- As a learner, I want the agent to treat its previous answer as revisable when
  my question exposes a weakness.
- As a learner, I want the agent to be brave enough to restructure and rewrite
  the artifact when my question shows the old structure is weak.
- As a learner, I want the agent to name tradeoffs, uncertainty, and alternative
  interpretations when my question has more than one reasonable frame.
- As a maintainer, I want update/create/restructure decisions to be testable
  from prompt contracts and artifact metadata.
