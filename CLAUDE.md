## Agent skills

### Issue tracker

Linear, team Boilerplate (`BOI`), through the `linear` MCP server. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical roles under the `triage` label group, names unchanged. See `docs/agents/triage-labels.md`.

### Domain docs

single-context, docs backend Notion (workspace Boilerplate, through the `notion-boilerplate` MCP server). See `docs/agents/domain.md`.

### Forge

GitHub at `github.com` through the `gh` CLI, repository `lcnascimento/boilerplate`. See `docs/agents/forge.md`.

### Environment

How to start the Project's dependencies and where its logs, metrics and traces live. See `docs/agents/environment.md`.

## Secrets

Secrets live only in Doppler and reach each process through its environment. See `docs/secrets.md`.

- Never read `.secrets` or `.env*` files; `.env.example` is the exception.
- A secret never gets a key in a `.env*` file or in a Tiltfile `serve_env`; `make env.check` enforces it. A versioned dev credential starts with `dev-only-`. See `CODING_STANDARDS.md`.
- Never run `doppler`, `tofu` or a `make iac.*` target that uses a credential (`iac.plan`, `iac.apply`, `iac.dev-token`): the human administrator runs them. `make iac.check` needs no credential.
- A `permissions.deny` rule or a hook that blocks you is the rule itself, not an obstacle to work around with another command, script or path.
- A real secret that reached your context must be rotated, even if it was never committed. Stop and tell the human; never rotate it yourself.
- When gitleaks blocks a commit:
  - never bypass it with `--no-verify`, `LEFTHOOK=0` or a `gitleaks:allow` comment, and never edit `.gitleaksignore`;
  - take the value out of the stage;
  - if the value may be real, treat it as leaked: stop and tell the human;
  - if it is a false positive, ask the human to record its fingerprint in `.gitleaksignore`.

## Language

- **Project language: Portuguese (pt-BR).** Talk to the user, and write documentation (`README.md`, `GLOSSARY.md` definitions, ADRs and other documents, Notion pages), issue tracker content and PR bodies, in Portuguese (pt-BR).
- **Code artifacts are English**: code, identifiers, comments, commit messages, branch names, PR titles, skills, `CLAUDE.md`/`AGENTS.md`, `CODING_STANDARDS.md` and `docs/agents/*`. `GLOSSARY.md` terms stay identical to the code identifiers.
