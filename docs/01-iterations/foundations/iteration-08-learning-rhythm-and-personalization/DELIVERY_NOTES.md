# Iteration 08 Delivery Notes

Status: delivered
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: actual iteration 08 delivery and validation record.

## Validation

- `go test ./backend-go/...` passes.
- `npm test -- --run` passes: 27 files, 89 tests.
- `npm run build` passes.
- Desktop visual checks cover the home rhythm and appearance settings in lychee paper, mountain mist, wisteria gray, and night ink.
- Night-mode visual checks also cover project reading surfaces.
- Representative normal-text contrast ratios exceed WCAG AA in every shipped palette.
- `config.local.example.json` parses successfully and `git diff --check` reports no whitespace errors.

## Residual risks

- GitHub currently reports that public Issue creation is restricted; repository settings must be changed by the maintainer before the Issue CTA can complete its intended flow.
- Effective reading is an intentionally conservative client signal, not precise time tracking.
- The repository's `npm run lint` is currently blocked by its existing ESLint 9 setup because no `eslint.config.js` is present; tests, type-checking, and production build remain green.
