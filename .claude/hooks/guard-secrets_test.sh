#!/usr/bin/env bash
set -uo pipefail

hook="$(dirname "$0")/guard-secrets.sh"
failures=0

pre_tool_use() {
	jq -cn --arg command "$1" '{
		session_id: "00000000-0000-0000-0000-000000000000",
		transcript_path: "/tmp/transcript.jsonl",
		cwd: "/repo",
		permission_mode: "default",
		hook_event_name: "PreToolUse",
		tool_name: "Bash",
		tool_input: {command: $command, description: "test"},
		tool_use_id: "toolu_test"
	}'
}

expect_raw() {
	local want=$1 input=$2 got
	printf '%s' "$input" | bash "$hook" 2>/dev/null
	got=$?
	if [ "$got" -ne "$want" ]; then
		printf 'FAIL: want exit %s, got %s: %s\n' "$want" "$got" "$input"
		failures=$((failures + 1))
	fi
}

allow() { expect_raw 0 "$(pre_tool_use "$1")"; }
deny() { expect_raw 2 "$(pre_tool_use "$1")"; }

deny 'env'
deny 'env -0'
deny 'env -u PATH'
deny 'env FOO=bar'
deny '/usr/bin/env'
deny 'env | grep KEY'
deny 'env > /tmp/out'
deny 'env env'
deny 'sudo env'
deny 'printenv'
deny 'printenv HOME'
deny 'export'
deny 'export -p'
deny 'declare -x'
deny 'declare -p'
deny 'set'
deny 'echo ok; set'
deny 'set | grep TOKEN'
deny 'ps e'
deny 'ps auxe'
deny 'ps eww 123'
deny 'ps -E'
deny 'ps -axE'
deny 'cat /proc/1/environ'
deny 'cat /proc/self/environ'
deny "tr '\\0' '\\n' < /proc/\$\$/environ"
deny "sh -c 'env'"
deny "bash -c \"printenv\""
deny "sh -c 'export -p'"
deny "python -c 'import os; os.system(\"env\")'"
deny 'x=$(env)'

deny 'cat .secrets'
deny 'cat ./.secrets'
deny 'cat /Users/me/repo/.secrets'
deny 'source .secrets'
deny 'grep TOKEN .secrets'
deny "sh -c 'cat .secrets'"
deny "python -c 'print(open(\".secrets\").read())'"
deny 'cp .secrets.example .secrets'
deny 'cat .secrets.bak'

deny 'ls ~/.doppler/'
deny 'cat /Users/me/.doppler/.doppler.yaml'
deny "sh -c 'cat \$HOME/.doppler/.doppler.yaml'"

deny 'doppler --version'
deny 'doppler secrets'
deny '/opt/homebrew/bin/doppler run -- true'
deny "sh -c 'doppler configure get token --plain'"
deny "python -c 'import subprocess; subprocess.run([\"doppler\", \"me\"])'"
deny 'which doppler'
deny 'tofu plan'
deny 'tofu -chdir=iac init'
deny "sh -c 'tofu apply'"
deny '/usr/local/bin/tofu version'

allow 'env FOO=bar true'
allow 'env FOO=bar BAZ=qux make test'
allow 'env -u FOO true'
allow '/usr/bin/env bash -c true'
allow 'export FOO=bar'
allow 'declare -x FOO=bar'
allow 'set -euo pipefail'
allow 'ps aux'
allow 'ps -ef'
allow 'ps -e'
allow 'ps -ef | grep tilt'
allow 'cat .secrets.example'
allow 'diff .secrets.example /dev/null'
allow 'cat .env.example'
allow 'make iac.check'
allow 'make test'
allow 'git status'
allow 'ls -la'
allow 'grep -r environment docs'
allow 'echo settings'

expect_raw 2 ''
expect_raw 2 'not json'
expect_raw 2 '{}'
expect_raw 2 '{"tool_name":"Bash","tool_input":{}}'
expect_raw 2 '{"tool_name":"Bash","tool_input":{"command":42}}'

if [ "$failures" -gt 0 ]; then
	printf 'guard-secrets: %d case(s) failed\n' "$failures"
	exit 1
fi
echo 'guard-secrets: all cases pass'
