# Contributing to Learn-With-AI

Thanks for helping improve this local learning lab.

## Report a problem

Use GitHub Issues for reproducible bugs and focused feature proposals. Include
the LLL version or commit, operating system, reproduction steps, expected
behavior, actual behavior, and screenshots or logs with secrets removed.

## Submit a change

1. Fork the repository and create a focused branch.
2. Read `AGENTS.md` and `README.md`.
3. Run `npm run check:fast` and `npm run build` before opening the PR.
4. Open a Pull Request explaining the learner-facing outcome and how you
   validated it (commands run, screenshots for UI changes).

Since v0.5.0 the maintainer keeps engineering documents (`docs/`, except the
release bundle and README images) and automated test suites in the local
working copy only; new files under those paths are ignored by Git. Describe
design notes and validation in the PR instead of adding them to the repository.

Do not commit `config.local.json`, API keys, generated run data, or private
learning projects. `npm run check:repo` verifies that every tracked path is
permitted by `.gitignore`; it is also part of every `check:*` quality gate.

Use one focused Conventional Commit message per independently reviewable
change: `type(scope): imperative summary`. Typical types are `feat`, `fix`,
`refactor`, `test`, `docs`, and `chore`. Stage explicit paths, inspect
`git diff --cached`, and include only source, tests, committed non-secret
configuration, product assets, or documentation that the change needs.
