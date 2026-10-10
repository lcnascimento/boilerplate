# Issue tracker: Linear

Issues and specs for this Project live in Linear, team **Boilerplate** (key `BOI`), in this Project's own Linear workspace. Use the `linear` MCP server's tools (`mcp__linear__*`) for all operations; never the forge's issue tracker. The server authenticates with `LINEAR_API_KEY` from this Project's `.env`, so it reaches only this Project's workspace.

## Conventions

- **Create an issue**: `save_issue` with `team: "Boilerplate"`, `title`, `description` (Markdown, literal newlines). Set `project` only when the work belongs to a Linear project (a finite deliverable); there is no catch-all project.
- **Read an issue**: `get_issue` with the identifier (e.g. `BOI-12`), plus `list_comments` for the thread.
- **List issues**: `list_issues` with `team`, `label`, `state`, `parentId`, `assignee` filters as needed.
- **Comment**: `save_comment` on the issue.
- **Labels**: `save_issue` with `addLabels` / `removeLabels`. Labels live at workspace level, lowercase. Grouped labels are named without a prefix: `type` (`bug`, `feature`, `improvement`, `chore`), `triage` (see `docs/agents/triage-labels.md`), `document` (one per Document type: `adr`, `tutorial`, `how-to`, `reference`, `explanation`). An issue that delivers a document carries `document:<type>` instead of a `type` label. If a label doesn't exist yet, create it at workspace level with `save_issue_label`.
- **Close**: `save_issue` with `state: "Done"` (or `"Canceled"` for won't-do), after commenting the reason.

Refer to issues by title in prose; the identifier (`BOI-12`) and URL ride inside the link.

## When a skill says "publish to the issue tracker"

Create a Linear issue in team Boilerplate.

## When a skill says "fetch the relevant ticket"

`get_issue` + `list_comments` on the identifier.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue with **sub-issues** as tickets.

- **Map**: one issue labelled `wayfinder:map`, holding the Destination / Notes / Decisions-so-far / Not-yet-specified / Out-of-scope body.
- **Child ticket**: a Linear sub-issue of the map (`save_issue` with `parentId: <map identifier>`). Labels: `wayfinder:<type>` (`research` / `prototype` / `grilling` / `task`), flat and prefixed. Create missing `wayfinder:*` labels at workspace level with `save_issue_label` on first use.
- **Document ticket**: a child labelled `document:<type>` delivers that document. Resolve it by writing the document with `foundation:documentation` into the docs backend (`docs/agents/domain.md`), and link the document from the resolution comment.
- **Blocking**: Linear's native **blocks / blocked by** relations, which are visible in the UI. Add an edge with `save_issue` on the child: `blockedBy: [<blocker identifier>]`. A ticket is unblocked when every blocker is in a completed or canceled state.
- **Frontier query**: `list_issues` with `parentId: <map>` and open states. Drop any with an assignee, or with an open blocker (check via `get_issue` relations). First in map order (creation order) wins.
- **Claim**: `save_issue` with `assignee: "me"`, the session's first write.
- **Resolve**: `save_comment` with the answer, then `save_issue` with `state: "Done"`, then append a context pointer (gist + link) to the map's Decisions-so-far (`save_issue` with a `patch` append on the map).
