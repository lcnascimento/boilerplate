#!/usr/bin/env bash
set -euo pipefail

grafana_url=${GRAFANA_URL:-http://localhost:3010}
admin_auth=${GRAFANA_ADMIN_AUTH:-admin:admin}
account=MCP
var=GRAFANA_SERVICE_ACCOUNT_TOKEN

fail() {
	echo "make grafana-token: $*" >&2
	exit 1
}

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)
cd "$root"

[[ -f .env ]] || cp .env.example .env

for _ in $(seq 60); do
	curl -fsS -o /dev/null "$grafana_url/api/health" 2>/dev/null && break
	sleep 1
done
curl -fsS -o /dev/null "$grafana_url/api/health" || fail "Grafana is not answering at $grafana_url; start it with tilt up"

current=$(sed -n "s/^$var=//p" .env | tail -1)
if [[ -n $current ]] &&
	[[ $(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $current" "$grafana_url/api/org") == 200 ]]; then
	echo "==> $var in .env is valid"
	exit 0
fi

admin() {
	curl -fsS -u "$admin_auth" -H 'Content-Type: application/json' "$@"
}

id=$(admin "$grafana_url/api/serviceaccounts/search?query=$account" |
	jq -r --arg name "$account" '.serviceAccounts[] | select(.name == $name) | .id')
if [[ -z $id ]]; then
	id=$(admin -X POST -d "{\"name\":\"$account\",\"role\":\"Viewer\"}" "$grafana_url/api/serviceaccounts" | jq -r .id)
fi

for token in $(admin "$grafana_url/api/serviceaccounts/$id/tokens" | jq -r '.[].id'); do
	admin -X DELETE -o /dev/null "$grafana_url/api/serviceaccounts/$id/tokens/$token"
done

key=$(admin -X POST -d "{\"name\":\"$account-$(date +%s)\"}" "$grafana_url/api/serviceaccounts/$id/tokens" | jq -r .key)
[[ -n $key && $key != null ]] || fail "Grafana returned no token"

tmp=$(mktemp)
KEY=$key awk -v var="$var" '
	$0 ~ "^" var "=" { if (!done) print var "=" ENVIRON["KEY"]; done = 1; next }
	{ print }
	END { if (!done) print var "=" ENVIRON["KEY"] }
' .env >"$tmp"
cat "$tmp" >.env
rm -f "$tmp"

echo "==> Wrote a new $var to .env; restart Claude Code so the grafana MCP server reads it"
