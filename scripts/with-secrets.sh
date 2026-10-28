#!/usr/bin/env bash
set -euo pipefail

secrets_file="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)/.secrets"
unset DOPPLER_TOKEN

token=
if [[ -f $secrets_file ]]; then
	while IFS= read -r line || [[ -n $line ]]; do
		[[ $line != DOPPLER_TOKEN=* ]] || token=${line#DOPPLER_TOKEN=}
	done <"$secrets_file"
fi

[[ -n $token ]] || exec "$@"

command=("$@")
offline() {
	echo "with-secrets: starting without the dev secrets of Doppler: $1" >&2
	exec "${command[@]}"
}

command -v doppler >/dev/null ||
	offline '.secrets has a DOPPLER_TOKEN but the doppler CLI is not installed'

DOPPLER_TOKEN=$token doppler run --no-fallback -- true >/dev/null ||
	offline 'doppler could not fetch them'

export DOPPLER_TOKEN=$token
exec doppler run --no-fallback -- env -u DOPPLER_TOKEN "$@"
