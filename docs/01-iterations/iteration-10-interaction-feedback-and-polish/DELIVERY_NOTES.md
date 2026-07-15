# Iteration 10 Delivery Notes

Status: delivered
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: delivery evidence and residual risk for iteration 10.

## Implemented

- Explicit discipline-overview rows and labeled move menus.
- In-app delete confirmation for folders, discipline projects, and learning units.
- Home navigation and learner-language cleanup.
- Flower newline rendering and content-idempotent activity events.
- Advisor missing-data guard and accessible waiting progress.
- Joined streak day label and white-first sky-blue theme.
- Game-like guided greenhouse with purpose copy, a three-step guide, collection
  progress, planting beds, observation details, responsive layout, and
  accessible interaction states.
- Bottom-edge text selections now use a viewport-level action toolbar that
  flips above the selection and clamps within narrow screens.

## Verification

- `npm run build`: passed.
- `npm run test -- --run`: passed (34 test files, 109 tests).
- `go test ./...`: passed for all Go packages.
- Browser smoke test: passed on desktop and a 700 px viewport. Verified the
  explicit discipline-overview row, Home navigation, joined streak label,
  a real greenhouse state with one flower and five petals, guide content,
  observation panel, and no horizontal overflow.

The frontend test run still prints React Router future-flag notices. They are
non-failing migration warnings and are not introduced by this slice.
