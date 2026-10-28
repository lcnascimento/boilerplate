# Issue tracker: Linear

Issues and specs for this Project live in Linear, team **Boilerplate** (key `BOI`), in this Project's own Linear workspace. Use the `linear` MCP server's tools (`mcp__linear__*`) for all operations; never the forge's issue tracker. The server authenticates with OAuth (see `docs/agents/environment.md`), and the sign-in picks the workspace.

## Conventions

- **Workspace check**: before the session's first write to Linear, call `get_workspace` and confirm it answers `Boilerplate`. When it answers another workspace, write nothing: ask the user to run `/mcp`, pick `linear`, choose "Clear authentication", then authenticate again choosing workspace `Boilerplate`.
- **Create an issue**: `save_issue` with `team: "Boilerplate"`, `title`, `description` (Markdown, literal newlines). Set `project` only when the work belongs to a Linear project (a finite deliverable); there is no catch-all project.
- **Read an issue**: `get_issue` with the identifier (e.g. `BOI-12`), plus `list_comments` for the thread.
- **List issues**: `list_issues` with `team`, `label`, `state`, `parentId`, `assignee` filters as needed.
- **Comment**: `save_comment` on the issue.
- **Labels**: `save_issue` with `addLabels` / `removeLabels`. Labels live at workspace level, lowercase. Grouped labels are named without a prefix: `type` (`bug`, `feature`, `improvement`, `chore`), `triage` (see `docs/agents/triage-labels.md`), `document` (one per Document type: `adr`, `tutorial`, `how-to`, `reference`, `explanation`). An issue that delivers a document carries `document:<type>` instead of a `type` label. If a label doesn't exist yet, create it at workspace level with `save_issue_label`.
- **Close**: `save_issue` with `state: "Done"` (or `"Canceled"` for won't-do), after commenting the reason.

Refer to issues by title in prose; the identifier (`BOI-12`) and URL ride inside the link.

## Work lifecycle

Skills that build a ticket (`implement`, `implement-spec`) name lifecycle roles. Each maps to a workflow status of team Boilerplate, set with `save_issue` and `state: "<status>"`:

| Role | Status |
| - | - |
| start | `In Progress` |
| hand off for review | `In Review` |
| close | `Done` |

- **Approval**: the merge of the work's PR is the user's approval. The PR body carries one `Closes <identifier>` line per issue it resolves, so Linear's forge integration closes each one on merge. When the user approves the work in the session instead, close each issue yourself.
- **Missing status**: statuses belong to each team. When `list_issue_statuses` for team Boilerplate has no `In Review` in any letter case, stop and ask the user to create it in Linear's UI (team Boilerplate, Settings, Workflow): typed `started`, placed after the team's other `started` statuses. The MCP server can't create statuses. Resume once it exists.

## When a skill says "publish to the issue tracker"

Create a Linear issue in team Boilerplate.

## When a skill says "fetch the relevant ticket"

`get_issue` + `list_comments` on the identifier.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **sub-issues** as tickets.

- **Map**: one issue labelled `wayfinder:map`, holding the Destination / Notes / Decisions-so-far / Not-yet-specified / Out-of-scope body.
- **Child ticket**: a Linear sub-issue of the map (`save_issue` with `parentId: <map identifier>`). Labels: `wayfinder:<type>` (`research` / `prototype` / `grilling` / `task`), flat and prefixed. Create missing `wayfinder:*` labels at workspace level with `save_issue_label` on first use.
- **Document ticket**: a child labelled `document:<type>` delivers that document. Resolve it by writing the document with `foundation:documentation` into the docs backend (`docs/agents/domain.md`), and link the document from the resolution comment.
- **Research ticket**: a child labelled `wayfinder:research` keeps its findings on the ticket, never in the repo. The subagent writes the `research` skill's Markdown file outside the repo (its scratchpad), then posts one resolution comment: the gist first, then the whole file, sources included. No `research/<name>` branch, commit or push: this replaces the throwaway branch of `/wayfinder`'s chart step.
- **Blocking**: Linear's native **blocks / blocked by** relations, which are visible in the UI. Add an edge with `save_issue` on the child: `blockedBy: [<blocker identifier>]`. A ticket is unblocked when every blocker is in a completed or canceled state.
- **Frontier query**: `list_issues` with `parentId: <map>` and open states. Drop any with an assignee, or with an open blocker (check via `get_issue` relations). First in map order (creation order) wins.
- **Claim**: `save_issue` with `assignee: "me"`, the session's first write.
- **Resolve**: `save_comment` with the answer, then `save_issue` with `state: "Done"`, then append a context pointer (gist + link) to the map's Decisions-so-far (`save_issue` with a `patch` append on the map).
