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
- Inspect: `gh pr view`; wait for checks as in [Waiting for checks](#waiting-for-checks).
- Merge only after review and green checks.

## CI

CI is defined in `.github/workflows/` and is the deterministic gate: a pull request merges only when it passes. No workflow has path filters: every run checks the whole repository. Read a failing run with `gh run view --log-failed`.

| Workflow           | Runs on                                                                                           | Checks                                                                             |
| ------------------ | ------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `quality_gate.yml` | `workflow_call` only                                                                              | jobs `format`, `lint`, `test` and `e2e`, in parallel                               |
| `pull_request.yml` | every pull request, draft or not                                                                  | `quality_gate / <job>` and `lint.breaking`                                         |
| `release.yml`      | pushes to `main`                                                                                  | `quality_gate / <job>`                                                             |
| `template.yml`     | pull requests (opened, synchronize, reopened, ready for review; drafts skip) and pushes to `main` | `A / make init`, `A / <job>`, `B / make clean-example + make init` and `B / <job>` |

Every job checks out the repository, runs the composite action `.github/actions/setup` (Go from `backend/go.work` with `make -C backend setup`, Node from `.nvmrc` with `pnpm install` at the root and in `frontend/`, and Playwright Chromium for `test` and `e2e`) and then root Makefile targets only. A check reproduces locally with the target of the same name:

| Job             | Locally                                                                                                       |
| --------------- | ------------------------------------------------------------------------------------------------------------- |
| `format`        | `make format.check` (refuses a dirty tree; fails if `make format` leaves a diff or an untracked file)         |
| `lint`          | `make lint`                                                                                                   |
| `test`          | `make test` (starts Postgres; needs Docker)                                                                   |
| `e2e`           | `make stack.up && make test.e2e; make stack.down` (needs Tilt)                                                |
| `lint.breaking` | `make lint.breaking` (`buf breaking` against `AGAINST`, default `origin/main`; CI passes the fetched PR base) |

`e2e` also installs Tilt, uploads the `tilt-logs` artifact on failure and the `playwright-report` artifact unless cancelled, and always runs `make stack.down`.

Concurrency: `pull_request` cancels the previous run of the same pull request on a new push; `release` never cancels; `template` cancels on pull requests and never on `main`.

`template` checks the template itself and runs only in the `lcnascimento/boilerplate` repository. Scenario `A` runs `make init NAME=acme-app`, scenario `B` runs `make clean-example` plus `make init`; each runs structural checks, then calls `quality_gate` with that command as its `bootstrap` input. Each `bootstrap` ends by committing its result, so `format.check` sees a clean tree. After `setup`, the gate passes `bootstrap` to the composite action `.github/actions/bootstrap`, which runs it, then `git clean -fdX`, then `setup` again, so the bootstrapped project is set up like a fresh one. `make init` deletes `template.yml`, so a generated project keeps `quality_gate`, `pull_request` and `release`.

The lefthook `pre-push` hook runs `make format.check` then `make lint`, the same targets as the `format` and `lint` jobs.

A pushed branch with no pull request runs no CI: checks start when a PR opens (`pull_request` at once, `template` once it is not a draft), and `release` and `template` run again after the merge to `main`.

### Waiting for checks

After each push, run `gh pr checks <n> --watch --fail-fast` once, in the background, and wait for it to exit. Never wait with a `sleep` loop and never poll its output.

The watch can exit on the previous commit's checks, so when it exits, read the runs of the pushed SHA with `gh run list --commit "$(git rev-parse HEAD)" --json databaseId,workflowName,status,conclusion`:

- Failed: a run has `conclusion` `failure`, or `cancelled` with no newer run of the same workflow. Read it with `gh run view <databaseId> --log-failed`, even while other runs are still in progress.
- Passed: there is at least one run, and every run is `completed` with `success` or `skipped`.
- Stale: anything else (no run yet, or runs in progress with none failed). Run the same watch in the background once more; if the result is stale again, stop and report.
