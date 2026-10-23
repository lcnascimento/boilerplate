#!/usr/bin/env bash
set -euo pipefail

self=scripts/clean-example.sh

example_paths=(
	backend/services/notes
	backend/services/audit/internal/transport/consumer/notes.go
	backend/services/iam/pkg/roles/notes_test.go
	backend/config/bruno/collections/Notes
	openapi/notes.yaml
	frontend/apps/web/app/\(dashboard\)/notes
	frontend/apps/web/actions/notes.ts
	frontend/apps/web/shared/_messages/notes.json
	frontend/apps/web/e2e/notes.spec.ts
	frontend/packages/api/src/notes
)

marker_re='example:(begin|end)'

fail() {
	echo "make clean-example: $*" >&2
	exit 1
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$root"
[[ $(git rev-parse --show-toplevel 2>/dev/null) == "$root" ]] &&
	git ls-files --error-unmatch -- "$self" >/dev/null 2>&1 ||
	fail "$root is not the template's git tree ($self is not tracked at its root)"

[[ -z $(git status --porcelain) ]] ||
	fail "the git tree must be clean: commit or stash your changes first"
for path in "${example_paths[@]}"; do
	[[ -n $(git ls-files -- "$path") ]] ||
		fail "$path not found: the example was already removed"
done

rm -rf "${example_paths[@]}"

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

gofmt -w backend
go -C backend/services/audit mod edit -droprequire=github.com/lcnascimento/boilerplate/backend/services/notes/pkg
make -C backend tidy
make -C backend o11y.generate
make -C backend iam.generate

drop_section() {
	HEADING=$2 perl -0pi -e 's/^\Q$ENV{HEADING}\E\n.*?(?=^## |\z)//ms; s/\n+\z/\n/' "$1"
}

drop_section README.md "## Removendo o exemplo"
perl -0pi -e 's/^\.PHONY: clean-example\nclean-example:\n(?:\t.*\n)*\n?//m' Makefile

if leftovers=$(git grep -nE "$marker_re" -- ":!$self"); then
	echo "$leftovers" >&2
	fail "the markers above are not well-formed fences; fix them, or undo with: git checkout . && git clean -fd"
fi

rm -f "$self"
rmdir scripts 2>/dev/null || true

pnpm install
pnpm -C frontend install
pnpm -C frontend format

cat <<'EOF'

Removed the notes example. Next steps:
  1. Review the diff: git status && git diff
  2. Commit: git add -A && git commit -m "chore: remove the notes example"
EOF
