#!/usr/bin/env bash
set -uo pipefail

block() {
	printf 'guard-secrets: %s. This is a project rule (CLAUDE.md, ## Secrets), not an obstacle to work around.\n' "$1" >&2
	exit 2
}

command=$(jq -er '.tool_input.command | strings' 2>/dev/null) ||
	block 'cannot read tool_input.command from the PreToolUse input'

word_boundary_start='(^|[^[:alnum:]_])'
word_boundary_end='($|[^[:alnum:]_])'

[[ ${command//.secrets.example/} == *.secrets* ]] && block 'the command cites .secrets'
[[ $command == *.doppler/* ]] && block 'the command cites ~/.doppler/'
[[ $command =~ ${word_boundary_start}doppler${word_boundary_end} ]] && block 'the command calls doppler'
[[ $command =~ ${word_boundary_start}tofu${word_boundary_end} ]] && block 'the command calls tofu'
[[ $command =~ ${word_boundary_start}printenv${word_boundary_end} ]] && block 'printenv dumps the environment'
[[ $command =~ /proc/[^[:space:]]*/environ ]] && block '/proc/*/environ dumps the environment'

only_options() {
	local arg
	for arg in "$@"; do
		[[ $arg == -* ]] || return 1
	done
}

check_segment() {
	local -a words
	read -ra words <<<"$1"
	set -- "${words[@]+"${words[@]}"}"

	while [ $# -gt 0 ]; do
		if [[ $1 =~ ^[[:alpha:]_][[:alnum:]_]*= ]]; then
			shift
			continue
		fi
		case "${1##*/}" in
		sudo | exec | command | builtin | nohup | time | nice | xargs)
			shift
			while [[ ${1-} == -* ]]; do shift; done
			;;
		env)
			shift
			while [ $# -gt 0 ]; do
				case "$1" in
				-u | -C | -P | -S) shift 2 ;;
				-* | *=*) shift ;;
				*) break ;;
				esac
			done
			[ $# -eq 0 ] && block 'env without a command dumps the environment'
			;;
		export | declare | typeset)
			shift
			only_options "$@" && block 'listing exported variables dumps the environment'
			return
			;;
		set)
			[ $# -eq 1 ] && block 'set without arguments dumps the environment'
			return
			;;
		ps)
			shift
			local arg
			for arg in "$@"; do
				[[ ($arg =~ ^[[:alpha:]]+$ && $arg == *e*) || ($arg =~ ^-[[:alpha:]]+$ && $arg == *E*) ]] &&
					block 'ps with the e modifier or -E shows process environments'
			done
			return
			;;
		*) return ;;
		esac
	done
}

while IFS= read -r segment; do
	check_segment "$segment"
done < <(printf '%s\n' "$command" | tr ";&|(){}\`'\"<>" '\n')

exit 0
