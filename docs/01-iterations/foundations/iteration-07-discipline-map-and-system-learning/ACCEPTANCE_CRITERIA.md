# Iteration 07 Acceptance Criteria

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: black-box acceptance contract for iteration 07.

- [US-07.1] Before creation, the learner can choose `学科地图` or `系统学习` and nothing is created until submission.
- [US-07.2] `问问 AI` is a visible icon-led action at the upper-right of `项目形态` and opens before title or motivation is complete.
- [US-07.2] AI advice supports temporary multi-turn clarification, can present a recommendation and trade-off, and is cleared when closed.
- [US-07.2] Requesting advice never persists the conversation, creates a project, or silently changes the selected type.
- [US-07.3] A discipline map opens directly to one learner-facing `学科总览`.
- [US-07.3] A discipline map does not show Intro, Explain, Practice, Extend, or Summary navigation.
- [US-07.3] Its project brief contains map purpose and optional scope notes, never an ability ladder, completion standard, active phase, or five-zone name.
- [US-07.3] The overview renders a Word-style table of contents from the agent-authored heading hierarchy, followed by the matching explanatory body.
- [US-07.4] The discipline map is rendered as a clickable sidebar folder title; it never appears as an uncategorized child project row.
- [US-07.4] Topics mentioned in the overview do not appear as sidebar entries or empty projects.
- [US-07.4] Other project progress does not rewrite the overview prose.
- [US-07.5] Each selectable key-concept or branch H3 has one adjacent inline `深入学习` action in the body; there is no top topic-card index or detached generic deep-dive action.
- [US-07.5] Activating that inline action opens an editable, prefilled confirmation form.
- [US-07.5] A normal system-learning project is created only after the learner confirms the form.
- [US-07.6] Map-origin creation stores an ordinary flat project with no parent relation and classifies it in the map-backed folder.
- [US-07.6] System-learning projects remain movable through the existing sidebar folder control.
- The create modal keeps its actions visible while long system-learning fields scroll; map creation hides and does not submit system-learning-only fields.
- Overview generation explicitly launches the selected native Agent CLI in a visible terminal and returns a session/run record; it never calls the HTTP model provider directly.
- After launch, the page says the Agent CLI was started rather than claiming hidden generation progress or a predicted duration.
- A write to the root overview artifact invalidates and refreshes the rendered overview.
- [US-07.7] Existing projects without type metadata continue to open with the five-zone workflow.
- [US-07.8, superseded by iteration 09] Weak or missing prerequisites now render generated summaries directly; no supplement action remains.
- Chinese user-facing labels do not expose internal filenames or protocol terms.
- Backend tests, frontend tests, and the production build pass before delivery is marked complete.
