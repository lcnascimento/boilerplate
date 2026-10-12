#!/usr/bin/env bash
# make clean-example: removes the `notes` example service.
#
# Deletes the example's own files, then every block fenced by begin/end
# comment lines (`# example:begin` ... `# example:end`, in any comment syntax)
# in the shared files, fences included. It runs before or after make init, or
# alone. It never commits nor installs anything, and it is not idempotent: to
# undo it before committing, run `git checkout . && git clean -fd`.
set -euo pipefail

self=scripts/clean-example.sh

# Whole files and folders of the example.
example_paths=(
	backend/services/notes
	backend/config/bruno/collections/Notes
	openapi/notes.yaml
	frontend/apps/web/app/\(dashboard\)/notes
	frontend/apps/web/actions/notes.ts
	frontend/apps/web/shared/_messages/notes.json
	frontend/apps/web/e2e/notes.spec.ts
	frontend/packages/api/src/notes
)

# Any begin/end marker, fence or not: none may survive the cleanup.
marker_re='example:(begin|end)'

fail() {
	echo "make clean-example: $*" >&2
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
[[ -z $(git status --porcelain) ]] ||
	fail "the git tree must be clean: commit or stash your changes first"
for path in "${example_paths[@]}"; do
	[[ -n $(git ls-files -- "$path") ]] ||
		fail "$path not found: the example was already removed"
done

rm -rf "${example_paths[@]}"

# Strips each fenced block. A fence is a line holding only a comment with the
# marker. When the block sat between two blank lines, one of them goes too.
git grep -lzE "$marker_re" -- ":!$self" | xargs -0 perl -0777 -i -ne '
	my ($in, $skip_blank, $begin_line, @out) = (0, 0, 0);
	my $fence = qr{^\s*(?:\#|//|--|;|<!--|/\*|\{/\*)\s*example:(begin|end)\s*(?:-->|\*/\}?)?\s*$};
	my $n = 0;
	for my $line (split /^/) {
		$n++;
		if ($line =~ $fence) {
			if ($1 eq "begin") {
				die "$ARGV:$n: example:begin inside another fence\n" if $in;
				($in, $begin_line) = (1, $n);
			} else {
				die "$ARGV:$n: example:end without example:begin\n" unless $in;
				$in = 0;
				$skip_blank = @out && $out[-1] =~ /^\s*$/;
			}
			next;
		}
		next if $in;
		if ($skip_blank) {
			$skip_blank = 0;
			next if $line =~ /^\s*$/;
		}
		push @out, $line;
	}
	die "$ARGV:$begin_line: example:begin without example:end\n" if $in;
	my $text = join "", @out;
	$text =~ s/\n\n+\z/\n/;
	print $text;
'

# drop_section <file> <heading>: removes a Markdown section, from its heading
# up to the next heading of the same level (or the end of the file).
drop_section() {
	HEADING=$2 perl -0pi -e 's/^\Q$ENV{HEADING}\E\n.*?(?=^## |\z)//ms; s/\n+\z/\n/' "$1"
}

# Template-only parts.
drop_section README.md "## Removendo o exemplo"
perl -0pi -e 's/^\.PHONY: clean-example\nclean-example:\n(?:\t.*\n)*\n?//m' Makefile

if leftovers=$(git grep -nE "$marker_re" -- ":!$self"); then
	echo "$leftovers" >&2
	fail "the markers above are not well-formed fences; fix them, or undo with: git checkout . && git clean -fd"
fi

# Self-delete. The Makefile and scripts/ go once init is gone too.
rm -f "$self"
grep -q '^[a-z][a-z-]*:' Makefile || rm -f Makefile
rmdir scripts 2>/dev/null || true

cat <<'EOF'

Removed the notes example. Next steps:
  1. Review the diff: git status && git diff
  2. Install the frontend dependencies and format the files the fences left
     behind: pnpm -C frontend install && pnpm -C frontend format
  3. Commit: git add -A && git commit -m "chore: remove the notes example"
EOF
