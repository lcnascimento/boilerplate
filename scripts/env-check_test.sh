#!/usr/bin/env bash
set -uo pipefail

check="$(cd "$(dirname "$0")" && pwd)/env-check.sh"
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
failures=0

repo() {
	local dir=$work/$1
	shift
	mkdir -p "$dir"
	git -C "$dir" init -q
	while [ $# -gt 0 ]; do
		mkdir -p "$dir/$(dirname "$1")"
		printf '%b' "$2" > "$dir/$1"
		shift 2
	done
	git -C "$dir" add -A
	printf '%s' "$dir"
}

expect() {
	local name=$1 want_status=$2 want_output=$3 dir=$4 got_output got_status
	got_output=$(bash "$check" "$dir" 2>&1)
	got_status=$?
	if [ "$got_status" -ne "$want_status" ] || [ "$got_output" != "$want_output" ]; then
		printf 'FAIL: %s\n  want exit %s, output:\n%s\n  got exit %s, output:\n%s\n' \
			"$name" "$want_status" "$want_output" "$got_status" "$got_output"
		failures=$((failures + 1))
	fi
}

expect 'a suspicious key without dev-only- fails' 1 \
	"backend/.env.example: POSTHOG_PROJECT_KEY looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)
web/.env.example: AUTH_TOKEN looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)
web/.env.example: DB_PASSWORD looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)
web/.env.example: CLIENT_SECRET_VALUE looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)" \
	"$(repo suspicious \
		backend/.env.example 'POSTHOG_HOST=\nPOSTHOG_PROJECT_KEY=\n' \
		web/.env.example 'AUTH_TOKEN=abc\nDB_PASSWORD=postgres\nCLIENT_SECRET_VALUE="real"\n')"

expect 'a suspicious key with dev-only- passes' 0 '' \
	"$(repo dev-only \
		web/.env.example 'BETTER_AUTH_SECRET=dev-only-secret\nAUDIT_IP_HASH_KEY="dev-only-key"\nIAM_TOKEN=dev-only-token\n')"

expect 'a key that does not look like a secret passes' 0 '' \
	"$(repo plain \
		web/.env.example '# API_KEY=commented\nPOSTHOG_HOST=\nKEYCLOAK_URL=http://localhost\nKEY_ID=1\n' \
		backend/.env.example 'POSTGRES_ADMIN_URL=postgres://postgres:postgres@localhost:5432/postgres\n')"

expect 'the root .env.example is ignored' 0 '' \
	"$(repo root \
		.env.example 'GRAFANA_SERVICE_ACCOUNT_TOKEN=\n' \
		web/.env.example 'POSTHOG_HOST=\n')"

expect 'an unversioned .env.example is ignored' 0 '' \
	"$(repo unversioned web/.env.example 'POSTHOG_HOST=\n'; printf 'API_KEY=\n' > "$work/unversioned/web/.env"; mkdir -p "$work/unversioned/new"; printf 'API_KEY=\n' > "$work/unversioned/new/.env.example")"

if [ "$failures" -gt 0 ]; then
	printf '%s env-check case(s) failed\n' "$failures"
	exit 1
fi
echo "env-check: all cases pass"
