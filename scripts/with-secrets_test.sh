#!/usr/bin/env bash
set -uo pipefail

wrapper="$(cd "$(dirname "$0")" && pwd)/with-secrets.sh"
failures=0
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# A checkout of its own, so the wrapper finds the .secrets of each case at its root.
setup_checkout() {
	rm -rf "$work/repo" "$work/tools" "$work/cli" "$work/out"
	mkdir -p "$work/repo/scripts" "$work/tools" "$work/cli" "$work/out"
	cp "$wrapper" "$work/repo/scripts/with-secrets.sh"
	local tool
	for tool in bash env true; do ln -s "$(type -P "$tool")" "$work/tools/$tool"; done

	cat >"$work/child" <<'EOF'
#!/usr/bin/env bash
printf 'argv=%s|' "$@" >"$OUT/child"
printf 'secret=%s token=%s\n' "${POSTHOG_PROJECT_KEY-unset}" "${DOPPLER_TOKEN-unset}" >>"$OUT/child"
EOF
	chmod +x "$work/child"

	cat >"$work/cli/doppler" <<'EOF'
#!/usr/bin/env bash
printf 'token=%s argv=%s\n' "${DOPPLER_TOKEN-unset}" "$*" >>"$OUT/doppler"
if [ -n "${FAKE_DOPPLER_FAIL-}" ]; then
	echo 'fake doppler: unable to fetch secrets' >&2
	exit 1
fi
while [ "$1" != -- ]; do shift; done
shift
export POSTHOG_PROJECT_KEY=from-doppler
exec "$@"
EOF
	chmod +x "$work/cli/doppler"
}

# run <case> <path>: runs the child through the wrapper with only the tools of <path> on PATH.
run() {
	local name=$1 path=$2
	shift 2
	env -i OUT="$work/out" PATH="$path" "$@" \
		"$work/repo/scripts/with-secrets.sh" "$work/child" serve 'a b' 2>"$work/out/stderr"
	got_status=$?
	got_child=$(cat "$work/out/child" 2>/dev/null || echo 'not started')
	got_doppler=$(cat "$work/out/doppler" 2>/dev/null || echo 'not called')
	got_stderr=$(cat "$work/out/stderr")
	case_name=$name
}

expect() {
	local field=$1 want=$2 got
	got=${!field}
	if [ "$got" != "$want" ]; then
		printf 'FAIL [%s] %s:\n  want: %s\n  got:  %s\n' "$case_name" "$field" "$want" "$got"
		failures=$((failures + 1))
	fi
}

offline_child='argv=serve|argv=a b|secret=unset token=unset'
warning='with-secrets: starting without the dev secrets of Doppler'

setup_checkout
run 'no .secrets' "$work/tools:$work/cli"
expect got_status 0
expect got_child "$offline_child"
expect got_doppler 'not called'
expect got_stderr ''

setup_checkout
run 'inherited DOPPLER_TOKEN, no .secrets' "$work/tools:$work/cli" DOPPLER_TOKEN=dp.st.inherited
expect got_child "$offline_child"
expect got_doppler 'not called'

setup_checkout
printf 'DOPPLER_TOKEN=\n' >"$work/repo/.secrets"
run 'empty token' "$work/tools:$work/cli"
expect got_status 0
expect got_child "$offline_child"
expect got_doppler 'not called'
expect got_stderr ''

setup_checkout
printf 'DOPPLER_TOKEN=dp.st.dev.fake\n' >"$work/repo/.secrets"
run 'token and CLI' "$work/tools:$work/cli"
expect got_status 0
expect got_child 'argv=serve|argv=a b|secret=from-doppler token=unset'
expect got_doppler "token=dp.st.dev.fake argv=run --no-fallback -- true
token=dp.st.dev.fake argv=run --no-fallback -- env -u DOPPLER_TOKEN $work/child serve a b"
expect got_stderr ''

setup_checkout
printf 'DOPPLER_TOKEN=dp.st.dev.fake\n' >"$work/repo/.secrets"
run 'token and failing CLI' "$work/tools:$work/cli" FAKE_DOPPLER_FAIL=1
expect got_status 0
expect got_child "argv=serve|argv=a b|secret=unset token=unset"
expect got_doppler 'token=dp.st.dev.fake argv=run --no-fallback -- true'
expect got_stderr "fake doppler: unable to fetch secrets
$warning: doppler could not fetch them"

setup_checkout
printf 'DOPPLER_TOKEN=dp.st.dev.fake\n' >"$work/repo/.secrets"
run 'token without CLI' "$work/tools"
expect got_status 0
expect got_child "$offline_child"
expect got_doppler 'not called'
expect got_stderr "$warning: .secrets has a DOPPLER_TOKEN but the doppler CLI is not installed"

if [ "$failures" -gt 0 ]; then
	printf 'with-secrets: %d check(s) failed\n' "$failures"
	exit 1
fi
echo 'with-secrets: all cases pass'
