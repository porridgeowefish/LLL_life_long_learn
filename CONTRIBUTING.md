# Contributing To LifeLongLearn

Thanks for helping improve this local learning lab.

## Report a problem

Use GitHub Issues for reproducible bugs and focused feature proposals. Include
the LLL version or commit, operating system, reproduction steps, expected
behavior, actual behavior, and screenshots or logs with secrets removed.

## Submit a change

1. Fork the repository and create a focused branch.
2. Read `AGENTS.md`, `docs/INDEX.md`, and the active iteration contract.
3. Keep code, API contract, tests, and delivery notes aligned in the same PR.
4. Run `go test ./backend-go/...`, `npm test` and `npm run build` in `frontend/`.
5. Open a Pull Request explaining the learner-facing outcome and validation.

Do not commit `config.local.json`, API keys, generated run data, or private
learning projects.
