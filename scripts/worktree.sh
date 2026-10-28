#!/usr/bin/env bash
set -euo pipefail

fail() {
	echo "make worktree: $*" >&2
	exit 1
}

usage() {
	fail "usage: $0 add <name> [base] [reset] | remove <name> [force]"
}

root=$(dirname "$(git rev-parse --path-format=absolute --git-common-dir)")
include="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)/.worktreeinclude"

valid_name() {
	[[ $1 =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]] && git check-ref-format --branch "$1" >/dev/null 2>&1 ||
		fail "name must be a branch name made of letters, digits, '.', '_' and '-' (e.g. name=boi-12), got '$1'"
}

registered() {
	git -C "$root" worktree list --porcelain | grep -qxF "worktree $1"
}

branch_exists() {
	git -C "$root" show-ref --verify --quiet "refs/heads/$1"
}

add() {
	local name=$1 base=$2 reset=$3 dir
	valid_name "$name"
	dir="$root/.claude/worktrees/$name"

	if [[ -z $base ]]; then
		git -C "$root" fetch origin main || fail "git fetch origin main failed; pass base=<branch> to work offline"
		base=origin/main
	fi
	git -C "$root" rev-parse --verify --quiet "$base^{commit}" >/dev/null || fail "base '$base' is not a commit"

	if registered "$dir"; then
		echo "==> Reusing $dir"
		if [[ -n $reset ]]; then
			echo "==> Resetting branch $name to $base"
			git -C "$dir" switch -C "$name" "$base"
		fi
	elif [[ -n $reset ]]; then
		git -C "$root" worktree add -B "$name" "$dir" "$base"
	else
		branch_exists "$name" && fail "branch '$name' already exists; pass reset=1 to recreate it from $base"
		git -C "$root" worktree add -b "$name" "$dir" "$base"
	fi

	local file
	while IFS= read -r file; do
		[[ -e $dir/$file ]] && continue
		echo "==> Copying $file"
		mkdir -p "$(dirname "$dir/$file")"
		cp -p "$root/$file" "$dir/$file"
	done < <(git -C "$root" ls-files --others --ignored --exclude-from="$include")

	make -C "$dir" setup
	echo "$dir"
}

remove() {
	local name=$1 force=$2 dir upstream
	valid_name "$name"
	dir="$root/.claude/worktrees/$name"

	if [[ -z $force ]] && branch_exists "$name"; then
		upstream=$(git -C "$root" rev-parse --verify --quiet "$name@{upstream}" || git -C "$root" rev-parse HEAD)
		git -C "$root" merge-base --is-ancestor "$name" "$upstream" ||
			fail "branch '$name' has unmerged commits; pass force=1 to delete it anyway"
	fi

	if registered "$dir"; then
		if [[ -n $force ]]; then
			git -C "$root" worktree remove --force "$dir"
		else
			git -C "$root" worktree remove "$dir" ||
				fail "$dir has uncommitted changes; pass force=1 to discard them"
		fi
	else
		echo "==> No worktree at $dir"
	fi

	if branch_exists "$name"; then
		git -C "$root" branch "$([[ -n $force ]] && echo -D || echo -d)" "$name"
	else
		echo "==> No branch $name"
	fi
}

case ${1:-} in
add) add "${2:-}" "${3:-}" "${4:-}" ;;
remove) remove "${2:-}" "${3:-}" ;;
*) usage ;;
esac
