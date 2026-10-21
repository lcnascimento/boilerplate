# Forge: GitHub (github.com)

This Project's code lives on GitHub at `github.com`. Use the `gh` CLI for every forge operation (pull requests, review, CI). Read the forge from this file; never infer it from `git remote`. Issues live in the issue tracker (`docs/agents/issue-tracker.md`), never in the forge.

- Repository: `lcnascimento/boilerplate`
- Default branch: `main`
- When a `gh` command runs outside this repo, set `GH_HOST=github.com`.

## Commits

Use Conventional Commits: `<type>(<optional scope>): <imperative summary>`, with `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci` or `build` as the type, and `!` or a `BREAKING CHANGE:` footer for breaking changes.

## GitHub pull requests

- Work on a branch, never on `main`.
- Open: `gh pr create --base main`, with the title following the commit convention and the body following the Project language.
- Inspect: `gh pr view`; wait for checks with `gh pr checks --watch`.
- Merge only after review and green checks.

## CI

CI is defined in `.github/workflows/` and is the deterministic gate: a pull request merges only when it passes. Read a failing run with `gh run view --log-failed`.

Six workflows check the repository as it is, each on the paths it covers (plus its own workflow file):

| Workflow              | Job            | Runs on                                                                    |
| --------------------- | -------------- | -------------------------------------------------------------------------- |
| `ci-backend.yml`      | `ci-backend`   | pull requests touching `backend/**`                                        |
| `ci-e2e.yml`          | `e2e`          | pull requests and pushes to `main` touching backend, frontend or Tiltfiles |
| `ci-frontend.yml`     | `frontend`     | pull requests and pushes to `main` touching `frontend/**`                  |
| `ci-proto.yml`        | `buf`          | pull requests and pushes to `main` touching proto or generated code        |
| `ci-openapi.yml`      | `vacuum`       | pull requests touching `openapi/**`                                        |
| `ci-suppressions.yml` | `suppressions` | every pull request and push to `main`                                      |

`ci-template.yml` runs on every push to any branch, with no path filter. It checks the template instead: it runs `make init` (scenario `A`) and `make clean-example` plus `make init` (scenario `B`), then calls the six workflows above on each result.

On a branch other than `main`, the six workflows check the repository as it is only on a pull request, so a pushed branch gets its full CI only once a PR is open, draft or not.
