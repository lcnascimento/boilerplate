#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
state=$root/.ephemeral
env_file=$root/.env.ephemeral
web_dir=$root/frontend/apps/web
compose_files=(-f "$root/backend/docker-compose.yml" -f "$root/backend/docker-compose.ephemeral.yml")

health_timeout=180
port_attempts=3

o11y_probes=(
	"alloy http://localhost:12345/-/ready"
	"prometheus http://localhost:9090/-/ready"
	"loki http://localhost:3100/ready"
	"tempo http://localhost:3200/ready"
	"pyroscope http://localhost:4040/ready"
)

fail() {
	echo "env: $*" >&2
	exit 1
}

say() {
	echo "==> $*"
}

env_name() {
	[[ $(git rev-parse --path-format=absolute --git-dir) != "$(git rev-parse --path-format=absolute --git-common-dir)" ]] ||
		fail "run it from a worktree: the main checkout is the \`local\` environment, under \`tilt up\`"
	local name
	name=$(basename "$root" | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-z0-9-]/-/g' -e 's/^-*//')
	case $name in
	'' | local | backend) fail "worktree directory '$(basename "$root")' gives the reserved environment name '$name'" ;;
	esac
	echo "$name"
}

compose() {
	docker compose -p "$env" "${compose_files[@]}" "$@"
}

free_port() {
	python3 -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1])'
}

ephemeral_var() {
	[[ ! -f $env_file ]] || sed -n "s/^$1=//p" "$env_file"
}

check_o11y() {
	local probe down=()
	for probe in "${o11y_probes[@]}"; do
		curl -s -o /dev/null --max-time 2 "${probe#* }" || down+=("${probe%% *}")
	done
	[[ ${#down[@]} -eq 0 ]] ||
		fail "the shared o11y stack is down (${down[*]}): run \`tilt up\` in the main checkout and wait for it to be ready"
}

# One line per process: <name> <service dir> <build target> <run target> <make variable holding its port>.
backend_processes() {
	local makefile service target suffix port_var
	for makefile in "$root"/backend/services/*/Makefile; do
		service=$(basename "$(dirname "$makefile")")
		for target in $(sed -n 's/^\(run\.[a-z.]*\):.*/\1/p' "$makefile"); do
			suffix=${target#run.}
			port_var=$(awk -v target="$target:" '$1 == target { found = 1; next } found { if (match($0, /PORT=\$\([A-Z_]+\)/)) print substr($0, RSTART + 7, RLENGTH - 8); exit }' "$makefile")
			[[ -n $port_var ]] || fail "$makefile: $target sets no *PORT=\$(VAR)"
			echo "$service-${suffix//./-} backend/services/$service build.$suffix $target $port_var"
		done
	done
}

port_key() {
	local key
	key=$(echo "$1" | tr '[:lower:]-' '[:upper:]_')
	echo "${key}_PORT"
}

service_vars() {
	local service=$1
	printf '%s\n' \
		"COMPOSE_PROJECT_NAME=$env" \
		"OTEL_RESOURCE_ATTRIBUTES=$OTEL_RESOURCE_ATTRIBUTES" \
		"DATABASE_URL=postgres://$service:$service@localhost:$POSTGRES_PORT/$service?sslmode=disable" \
		"POSTGRES_ADMIN_URL=$POSTGRES_ADMIN_URL" \
		"KAFKA_BROKERS=$KAFKA_BROKERS" \
		"SCHEMA_REGISTRY_URL=$SCHEMA_REGISTRY_URL" \
		"AUTH_JWKS_URL=$WEB_URL/api/auth/jwks" \
		"AUTH_ISSUER=$WEB_URL"
}

service_make() {
	local service=$1
	shift
	local vars=()
	while IFS= read -r var; do vars+=("$var"); done < <(service_vars "$service")
	make -s -C "$root/backend/services/$service" "$@" "${vars[@]}"
}

web_vars() {
	printf '%s\n' \
		"IAM_DATABASE_URL=$IAM_DATABASE_URL" \
		"BETTER_AUTH_URL=$WEB_URL" \
		"OTEL_SERVICE_NAME=web" \
		"OTEL_RESOURCE_ATTRIBUTES=$OTEL_RESOURCE_ATTRIBUTES" \
		"NEXT_OTEL_FETCH_DISABLED=1"
	# example:begin
	echo "NOTES_API_URL=$NOTES_API_URL"
	# example:end
}

with_web_env() {
	local vars=()
	while IFS= read -r var; do vars+=("$var"); done < <(web_vars)
	(
		set -a
		. "$web_dir/.env.local"
		set +a
		cd "$web_dir"
		exec env "${vars[@]}" "$@"
	)
}

start_containers() {
	local attempt=1 out
	EPHEMERAL_KAFKA_PORT=$(ephemeral_var KAFKA_PORT)
	[[ -n $EPHEMERAL_KAFKA_PORT ]] || EPHEMERAL_KAFKA_PORT=$(free_port)
	export EPHEMERAL_KAFKA_PORT
	until out=$(compose up -d --wait postgres kafka schema-registry 2>&1); do
		if [[ $attempt -lt $port_attempts ]] && grep -qiE 'port is already allocated|address already in use' <<<"$out"; then
			EPHEMERAL_KAFKA_PORT=$(free_port)
			attempt=$((attempt + 1))
			continue
		fi
		echo "$out" >&2
		fail "docker compose up failed for project $env"
	done
}

container_port() {
	local address
	address=$(compose port "$1" "$2")
	echo "${address##*:}"
}

read_container_ports() {
	POSTGRES_PORT=$(container_port postgres 5432)
	KAFKA_PORT=$EPHEMERAL_KAFKA_PORT
	SCHEMA_REGISTRY_PORT=$(container_port schema-registry 8081)
	OTEL_RESOURCE_ATTRIBUTES=$(sed -n 's/^OTEL_RESOURCE_ATTRIBUTES=//p' "$root/backend/.env" |
		sed "s/deployment\.environment=[^,]*/deployment.environment=$env/")
	POSTGRES_ADMIN_URL=postgres://postgres:postgres@localhost:$POSTGRES_PORT/postgres?sslmode=disable
	IAM_DATABASE_URL=postgres://iam:iam@localhost:$POSTGRES_PORT/iam?sslmode=disable
	KAFKA_BROKERS=localhost:$KAFKA_PORT
	SCHEMA_REGISTRY_URL=http://localhost:$SCHEMA_REGISTRY_PORT
}

allocate_process_ports() {
	local name _rest
	WEB_PORT=$(free_port)
	WEB_URL=http://localhost:$WEB_PORT
	BETTER_AUTH_URL=$WEB_URL
	while read -r name _rest; do
		printf -v "$(port_key "$name")" '%s' "$(free_port)"
	done < <(backend_processes)
	# example:begin
	NOTES_URL=http://localhost:$NOTES_API_FRONTEND_PORT
	NOTES_API_URL=$NOTES_URL
	# example:end
}

write_env_file() {
	local name _rest keys=(ENV_NAME COMPOSE_PROJECT_NAME OTEL_RESOURCE_ATTRIBUTES POSTGRES_PORT KAFKA_PORT SCHEMA_REGISTRY_PORT WEB_PORT)
	while read -r name _rest; do keys+=("$(port_key "$name")"); done < <(backend_processes)
	keys+=(WEB_URL BETTER_AUTH_URL IAM_DATABASE_URL POSTGRES_ADMIN_URL KAFKA_BROKERS SCHEMA_REGISTRY_URL)
	# example:begin
	keys+=(NOTES_URL NOTES_API_URL)
	# example:end
	ENV_NAME=$env
	COMPOSE_PROJECT_NAME=$env
	local key
	for key in "${keys[@]}"; do echo "$key=${!key}"; done >"$env_file"
}

bootstrap() {
	local service
	say "postgres-extensions"
	compose run --rm --no-deps postgres-extensions >/dev/null
	for service in "$root"/backend/services/*/; do
		service=$(basename "$service")
		say "$service: db.setup db.migrate kafka.topics"
		service_make "$service" db.setup db.migrate kafka.topics
		if grep -q '^schemas.register:' "$root/backend/services/$service/Makefile"; then
			say "$service: schemas.register"
			service_make "$service" schemas.register
		fi
	done
	say "iam-seed"
	with_web_env pnpm --filter @boilerplate/web iam:seed
}

build_processes() {
	local name dir build _run _port
	while read -r name dir build _run _port; do
		say "build $name"
		make -s -C "$root/$dir" "$build"
	done < <(backend_processes)
}

# Each process leads its own process group, so stopping it also stops what `make` or `pnpm` started.
spawn() {
	local name=$1
	shift
	set -m
	(
		trap '' HUP
		"$@"
	) </dev/null >"$state/$name.log" 2>&1 &
	echo $! >"$state/$name.pid"
	set +m
}

start_processes() {
	local name dir _build run port_var service vars port_key
	mkdir -p "$state"
	while read -r name dir _build run port_var; do
		service=$(basename "$dir")
		vars=()
		while IFS= read -r var; do vars+=("$var"); done < <(service_vars "$service")
		port_key=$(port_key "$name")
		vars+=("$port_var=${!port_key}")
		spawn "$name" make -s -C "$root/$dir" "$run" "${vars[@]}"
	done < <(backend_processes)
	spawn web with_web_env pnpm exec next dev --port "$WEB_PORT"
}

pid_of() {
	local pid
	[[ -f $state/$1.pid ]] || return 1
	pid=$(cat "$state/$1.pid")
	[[ $(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ') == "$pid" ]] || return 1
	echo "$pid"
}

stop_processes() {
	local pidfile name pid _
	[[ -d $state ]] || return 0
	for pidfile in "$state"/*.pid; do
		[[ -f $pidfile ]] || continue
		name=$(basename "$pidfile" .pid)
		if pid=$(pid_of "$name"); then
			say "stop $name"
			kill -TERM -- "-$pid" 2>/dev/null || true
			for _ in $(seq 20); do
				kill -0 -- "-$pid" 2>/dev/null || break
				sleep 0.5
			done
			kill -KILL -- "-$pid" 2>/dev/null || true
		fi
		rm -f "$pidfile"
	done
}

health_url() {
	case $1 in
	web) echo "$WEB_URL/" ;;
	*)
		local port_key
		port_key=$(port_key "$1")
		echo "http://localhost:${!port_key}/health"
		;;
	esac
}

healthy() {
	local code
	code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$(health_url "$1")" 2>/dev/null) || return 1
	case $1 in
	web) [[ $code -ge 200 && $code -lt 500 ]] ;;
	*) [[ $code == 200 ]] ;;
	esac
}

# Returns 2 when a process lost its port to another one between allocation and bind.
await_health() {
	local names=(web) name _rest pending deadline=$((SECONDS + health_timeout))
	while read -r name _rest; do names+=("$name"); done < <(backend_processes)
	while :; do
		pending=()
		for name in "${names[@]}"; do
			healthy "$name" && continue
			if ! pid_of "$name" >/dev/null; then
				if grep -qE 'address already in use|EADDRINUSE' "$state/$name.log"; then return 2; fi
				echo "env: $name exited before it was healthy; the end of $state/$name.log:" >&2
				tail -n 30 "$state/$name.log" >&2
				return 1
			fi
			pending+=("$name")
		done
		[[ ${#pending[@]} -gt 0 ]] || return 0
		if [[ $SECONDS -ge $deadline ]]; then
			echo "env: not healthy after ${health_timeout}s: ${pending[*]} (logs in $state)" >&2
			return 1
		fi
		sleep 1
	done
}

up() {
	check_o11y
	stop_processes
	say "containers of project $env"
	start_containers
	read_container_ports
	allocate_process_ports
	write_env_file
	bootstrap
	build_processes
	local attempt status
	for attempt in $(seq "$port_attempts"); do
		say "start processes (attempt $attempt)"
		start_processes
		status=0
		await_health || status=$?
		[[ $status -eq 0 ]] && break
		stop_processes
		[[ $status -eq 2 && $attempt -lt $port_attempts ]] || fail "environment $env did not come up"
		allocate_process_ports
		write_env_file
	done
	say "environment $env is up"
	cat "$env_file"
}

down() {
	stop_processes
	say "remove project $env"
	(cd / && docker compose -p "$env" down -v --remove-orphans)
	rm -rf "$state" "$env_file"
}

status() {
	if [[ -f $env_file ]]; then cat "$env_file"; else echo "no $env_file: run make env.up"; fi
	echo
	local pidfile name
	for pidfile in "$state"/*.pid; do
		[[ -f $pidfile ]] || continue
		name=$(basename "$pidfile" .pid)
		if pid_of "$name" >/dev/null; then echo "$name: running (pid $(cat "$pidfile"))"; else echo "$name: stopped"; fi
	done
	echo
	docker compose -p "$env" ps -a
}

prune() {
	local project dir
	docker ps -a --filter label=com.boilerplate.ephemeral=true \
		--format '{{.Label "com.docker.compose.project"}} {{.Label "com.docker.compose.project.working_dir"}}' |
		sort -u |
		while read -r project dir; do
			[[ -n $project && $project != backend ]] || continue
			[[ -d $dir ]] && continue
			say "remove project $project (its worktree $dir is gone)"
			(cd / && docker compose -p "$project" down -v --remove-orphans)
		done
}

case ${1:-} in
up) env=$(env_name) && up ;;
down) env=$(env_name) && down ;;
status) env=$(env_name) && status ;;
prune) prune ;;
*) fail "usage: env.sh up | down | status | prune" ;;
esac
