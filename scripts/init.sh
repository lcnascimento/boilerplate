#!/usr/bin/env bash
set -euo pipefail

marker=boilerplate
title_marker=Boilerplate
self=scripts/init.sh

fail() {
	echo "make init: $*" >&2
	exit 1
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$root"
[[ $(git rev-parse --show-toplevel 2>/dev/null) == "$root" ]] &&
	git ls-files --error-unmatch -- "$self" >/dev/null 2>&1 ||
	fail "$root is not the template's git tree ($self is not tracked at its root)"

name=${1:-}
[[ $name =~ ^[a-z][a-z0-9-]*$ ]] ||
	fail "NAME must match ^[a-z][a-z0-9-]*\$ (e.g. make init NAME=my-app), got '$name'"
[[ -z $(git status --porcelain) ]] ||
	fail "the git tree must be clean: commit or stash your changes first"
git grep -qi "$marker" -- ":!$self" ||
	fail "no '$marker' marker left: this project was already initialized"

title=$(perl -e 'print join " ", map { ucfirst } grep { length } split /-/, $ARGV[0]' "$name")

drop_section() {
	HEADING=$2 perl -0pi -e 's/^\Q$ENV{HEADING}\E\n.*?(?=^## |\z)//ms; s/\n+\z/\n/' "$1"
}

rm -rf docs/agents
drop_section CLAUDE.md "## Agent skills"

rm -f .github/workflows/template.yml
drop_section README.md "## Como instanciar"
perl -0pi -e 's/^\.PHONY: init\ninit:\n(?:\t.*\n)*\n?//m' Makefile

git rm -rq --ignore-unmatch -- iac/global/terraform.tfstate iac/environments/terraform.tfstate.d iac/environments/terraform.tfvars

proto_name=${name//-/_}
while IFS= read -r dir; do
	git mv "$dir/$marker" "$dir/$proto_name"
done < <(git ls-files -- '*.proto' | sed -n "s#^\(.*/proto\)/$marker/.*#\1#p" | sort -u)
{ git grep -lzIE "$marker[./][a-z_]+[./]v[0-9]" -- ":!$self" || true; } |
	NAME=$proto_name xargs -0 perl -pi -e 's/(?<![\w\/.-])boilerplate(?=[.\/][a-z_]+[.\/]v[0-9])/$ENV{NAME}/g'

git grep -lzI -e "$marker" -e "$title_marker" -- ":!$self" |
	NAME=$name TITLE=$title xargs -0 perl -pi -e 's/boilerplate/$ENV{NAME}/g; s/Boilerplate/$ENV{TITLE}/g'

if [[ -n $(git ls-files -- '*buf.gen.yaml') ]]; then
	make -C backend -s proto.tools proto.generate
fi

if leftovers=$(git grep -ni "$marker" -- ":!$self"); then
	echo "$leftovers" >&2
	fail "the lines above still carry the marker in another casing; fix them, or undo with: git checkout . && git clean -fd"
fi

rm -f "$self"
rmdir scripts 2>/dev/null || true

pnpm install
pnpm -C frontend install
pnpm -C frontend format

cat <<EOF

Initialized "$title" ($name). Next steps:
  1. Review the diff: git status && git diff
  2. Run the ai-foundation setup for this project (Linear, Notion, forge).
  3. Commit: git add -A && git commit -m "chore: initialize $name from the template"
  4. When the project needs its first secret, bootstrap Doppler: README.md, section "## Doppler".
EOF
