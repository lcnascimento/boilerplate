#!/usr/bin/env bash
set -euo pipefail

cd "${1:-.}"

status=0
while IFS= read -r file; do
	awk -v file="$file" -v tiltfile="$([[ $file == *Tiltfile ]] && echo 1)" -v q="'" '
		function check(key, value) {
			sub(/^[[:space:]]+/, "", value)
			sub("^[\"" q "]", "", value)
			if ((key ~ /SECRET|TOKEN|PASSWORD/ || key ~ /_KEY$/) && value !~ /^dev-only-/) {
				printf "%s in %s looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)\n", key, file
				found = 1
			}
		}
		!tiltfile && /^[A-Za-z_][A-Za-z0-9_]*=/ {
			check(substr($0, 1, index($0, "=") - 1), substr($0, index($0, "=") + 1))
		}
		tiltfile && !/^[[:space:]]*#/ {
			quote = "[\"" q "]"
			rest = $0
			while (match(rest, quote "[A-Za-z_][A-Za-z0-9_]*" quote "[[:space:]]*\\]?[[:space:]]*[:=]")) {
				entry = substr(rest, RSTART + 1, RLENGTH - 1)
				rest = substr(rest, RSTART + RLENGTH)
				key = substr(entry, 1, match(entry, quote) - 1)
				if (rest !~ /^=/) check(key, rest)
			}
		}
		END { exit found }
	' "$file" || status=1
done < <(git ls-files -- '*.env.example' ':!:.env.example' Tiltfile '*/Tiltfile')

exit "$status"
