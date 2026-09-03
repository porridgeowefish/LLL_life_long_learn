# Iteration 10 Acceptance Criteria

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: black-box acceptance for iteration 10.

1. A discipline folder expands to an explicit `××学科总览` link plus its learning units; the folder title only expands or collapses.
2. A learning-unit menu labels its destination list `移动到文件夹`; discipline projects and learning units expose appropriately named delete actions.
3. Folder, discipline-project, and learning-unit deletion use an in-app dialog; cancelling performs no deletion.
4. The top bar contains one `主页` entry and no duplicate `项目` entry to the same destination.
5. The greenhouse includes purpose guidance, collection/growth feedback, an empty state, and responsive game-like presentation.
6. Flower content displays saved line breaks. Each distinct saved flower representation creates one `extend-flower` activity event; saving identical content again does not duplicate it.
7. The project-type advisor request omits unavailable learner-level and completion-standard fields, and its prompt treats missing fields as unknown.
8. While advice is pending, an accessible indeterminate progress indicator and reassuring status text are visible.
9. No learner-facing `slugify` term remains in project creation.
10. Streak count and `天` read as one visual unit, and the former purple theme is presented as a white-first sky-blue theme.
11. Selecting text near any viewport edge keeps the selection action toolbar visible; near the bottom it opens above the selection instead of being clipped.
