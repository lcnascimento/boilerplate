#!/usr/bin/env bash
# make init NAME=<name>: turns the template into a project named <name>.
#
# Replaces the marker in every tracked file (`boilerplate` -> <name>,
# `Boilerplate` -> its title), drops the template's agent configuration and
# deletes the template-only parts, this script included. It never commits nor
# installs anything, and it is not idempotent: to undo it before committing,
# run `git checkout . && git clean -fd`.
set -euo pipefail

marker=boilerplate
title_marker=Boilerplate
self=scripts/init.sh

fail() {
	echo "make init: $*" >&2
	exit 1
}

# Work on the tree this script belongs to, never on the caller's cwd, and
# refuse a stray copy that is not tracked at the root of its own git tree.
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$root"
[[ $(git rev-parse --show-toplevel 2>/dev/null) == "$root" ]] &&
	git ls-files --error-unmatch -- "$self" >/dev/null 2>&1 ||
	fail "$root is not the template's git tree ($self is not tracked at its root)"

# Preconditions, all checked before any file is touched.
name=${1:-}
[[ $name =~ ^[a-z][a-z0-9-]*$ ]] ||
	fail "NAME must match ^[a-z][a-z0-9-]*\$ (e.g. make init NAME=my-app), got '$name'"
[[ -z $(git status --porcelain) ]] ||
	fail "the git tree must be clean: commit or stash your changes first"
git grep -qi "$marker" -- ":!$self" ||
	fail "no '$marker' marker left: this project was already initialized"

title=$(perl -e 'print join " ", map { ucfirst } grep { length } split /-/, $ARGV[0]' "$name")

# drop_section <file> <heading>: removes a Markdown section, from its heading
# up to the next heading of the same level (or the end of the file).
drop_section() {
	HEADING=$2 perl -0pi -e 's/^\Q$ENV{HEADING}\E\n.*?(?=^## |\z)//ms; s/\n+\z/\n/' "$1"
}

# The template's agent configuration points at its own Linear and Notion.
rm -rf docs/agents
drop_section CLAUDE.md "## Agent skills"

# Template-only parts.
rm -f .github/workflows/ci-template.yml
drop_section README.md "## Como instanciar"
perl -0pi -e 's/^\.PHONY: init\ninit:\n(?:\t.*\n)*\n?//m' Makefile

# The marker, in every tracked text file still present.
git grep -lzI -e "$marker" -e "$title_marker" -- ":!$self" |
	NAME=$name TITLE=$title xargs -0 perl -pi -e 's/boilerplate/$ENV{NAME}/g; s/Boilerplate/$ENV{TITLE}/g'

if leftovers=$(git grep -ni "$marker" -- ":!$self"); then
	echo "$leftovers" >&2
	fail "the lines above still carry the marker in another casing; fix them, or undo with: git checkout . && git clean -fd"
fi

# Self-delete. The Makefile and scripts/ go once clean-example is gone too.
rm -f "$self"
grep -q '^[a-z][a-z-]*:' Makefile || rm -f Makefile
rmdir scripts 2>/dev/null || true

cat <<EOF

Initialized "$title" ($name). Next steps:
  1. Review the diff: git status && git diff
  2. Install the dependencies and re-format the frontend (the new name changes
     line lengths): pnpm install && pnpm -C frontend install && pnpm -C frontend format
  3. Run the ai-foundation setup for this project (Linear, Notion, forge).
  4. Commit: git add -A && git commit -m "chore: initialize $name from the template"
EOF
