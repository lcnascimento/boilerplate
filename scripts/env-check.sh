#!/usr/bin/env bash
set -euo pipefail

cd "${1:-.}"

status=0
while IFS= read -r file; do
	awk -v file="$file" '
		/^[A-Za-z_][A-Za-z0-9_]*=/ {
			key = substr($0, 1, index($0, "=") - 1)
			value = substr($0, index($0, "=") + 1)
			gsub("^[\"\047]", "", value)
			if ((key ~ /SECRET|TOKEN|PASSWORD/ || key ~ /_KEY$/) && value !~ /^dev-only-/) {
				printf "%s: %s looks like a secret; a secret lives only in Doppler, and a versioned dev credential starts with dev-only- (docs/secrets.md)\n", file, key
				found = 1
			}
		}
		END { exit found }
	' "$file" || status=1
done < <(git ls-files -- '*.env.example' ':!:.env.example')

exit "$status"
