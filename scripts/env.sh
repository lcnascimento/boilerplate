#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
state=$root/.ephemeral
env_file=$root/.env.ephemeral
web_dir=$root/frontend/apps/web
compose_files=(-f "$root/backend/docker-compose.yml" -f "$root/backend/docker-compose.ephemeral.yml")

health_timeout=180
port_attempts=3
port_conflict=2
ephemeral_label=com.boilerplate.ephemeral=true
command=${1:-}

o11y_probes=(
	"alloy http://localhost:12345/-/ready"
	"prometheus http://localhost:9090/-/ready"
	"loki http://localhost:3100/ready"
	"tempo http://localhost:3200/ready"
	"pyroscope http://localhost:4040/ready"
)

fail() {
	echo "make env.$command: $*" >&2
	exit 1
}

say() {
	echo "==> $*"
}

project_name() {
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
	docker compose -p "$project" "${compose_files[@]}" "$@"
}

remove_project() {
	(cd / && docker compose -p "$1" down -v --remove-orphans)
}

free_port() {
	python3 -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1])'
}

running_kafka_port() {
	local id address
	id=$(docker ps -q --filter "label=com.docker.compose.project=$project" --filter label=com.docker.compose.service=kafka)
	[[ -n $id ]] || return 0
	address=$(docker port "$id" 9092 | head -1)
	echo "${address##*:}"
}

check_o11y() {
	local probe down=()
	for probe in "${o11y_probes[@]}"; do
		curl -s -o /dev/null --max-time 2 "${probe#* }" || down+=("${probe%% *}")
	done
	[[ ${#down[@]} -eq 0 ]] ||
		fail "the shared o11y stack is down (${down[*]}): run \`tilt up\` in the main checkout and wait for it to be ready"
}

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
	local service=$1 database_key
	database_key="$(echo "$service" | tr '[:lower:]' '[:upper:]')_DATABASE_URL"
	printf '%s\n' \
		"COMPOSE_PROJECT_NAME=$project" \
		"OTEL_RESOURCE_ATTRIBUTES=$OTEL_RESOURCE_ATTRIBUTES" \
		"$database_key=postgres://$service:$service@localhost:$POSTGRES_PORT/$service?sslmode=disable" \
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
		"BETTER_AUTH_URL=$BETTER_AUTH_URL" \
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
	EPHEMERAL_KAFKA_PORT=$(running_kafka_port)
	[[ -n $EPHEMERAL_KAFKA_PORT ]] || EPHEMERAL_KAFKA_PORT=$(free_port)
	export EPHEMERAL_KAFKA_PORT
	until out=$(compose up -d --wait postgres kafka schema-registry 2>&1); do
		if [[ $attempt -lt $port_attempts ]] && grep -qiE 'port is already allocated|address already in use' <<<"$out"; then
			EPHEMERAL_KAFKA_PORT=$(free_port)
			attempt=$((attempt + 1))
			continue
		fi
		echo "$out" >&2
		fail "docker compose up failed for project $project"
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
		sed "s/deployment\.environment=[^,]*/deployment.environment=$project/")
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
	ENV_NAME=$project
	COMPOSE_PROJECT_NAME=$project
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

# Leading its own process group tells this process apart from an unrelated one that reused its PID.
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
	local name dir _build run port_var service vars key
	mkdir -p "$state"
	while read -r name dir _build run port_var; do
		service=$(basename "$dir")
		vars=()
		while IFS= read -r var; do vars+=("$var"); done < <(service_vars "$service")
		key=$(port_key "$name")
		vars+=("$port_var=${!key}")
		spawn "$name" make -s -C "$root/$dir" "$run" "${vars[@]}"
	done < <(backend_processes)
	spawn web with_web_env "$root/scripts/with-secrets.sh" pnpm exec next dev --port "$WEB_PORT"
}

pid_of() {
	local pid
	[[ -f $state/$1.pid ]] || return 1
	pid=$(cat "$state/$1.pid")
	[[ $(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ') == "$pid" ]] || return 1
	echo "$pid"
}

spawned_processes() {
	local pidfile
	for pidfile in "$state"/*.pid; do
		[[ ! -f $pidfile ]] || basename "$pidfile" .pid
	done
}

descendants() {
	local child
	for child in $(ps -axo pid=,ppid= | awk -v parent="$1" '$2 == parent { print $1 }'); do
		echo "$child"
		descendants "$child"
	done
}

any_alive() {
	local pid
	for pid in "$@"; do
		! kill -0 "$pid" 2>/dev/null || return 0
	done
	return 1
}

# pnpm starts `next dev` in a process group of its own, so the whole tree is signalled, collected before it reparents.
stop_processes() {
	local name pid tree _
	for name in $(spawned_processes); do
		if pid=$(pid_of "$name"); then
			say "stop $name"
			tree=$(echo "$pid" && descendants "$pid")
			kill -TERM $tree 2>/dev/null || true
			for _ in $(seq 20); do
				any_alive $tree || break
				sleep 0.5
			done
			kill -KILL $tree 2>/dev/null || true
		fi
		rm -f "$state/$name.pid"
	done
}

descends_from() {
	local pid=$1
	while [[ -n $pid && $pid -gt 1 ]]; do
		[[ $pid != "$2" ]] || return 0
		pid=$(ps -o ppid= -p "$pid" | tr -d ' ')
	done
	return 1
}

# The port may have gone to another environment's process between allocation and bind, and that one answers too.
listens_on() {
	local leader pid
	leader=$(pid_of "$1") || return 1
	for pid in $(lsof -nP -t -iTCP:"$2" -sTCP:LISTEN 2>/dev/null); do
		! descends_from "$pid" "$leader" || return 0
	done
	return 1
}

healthy() {
	local key port path=/health
	key=$(port_key "$1")
	port=${!key}
	[[ $1 != web ]] || path=/
	listens_on "$1" "$port" &&
		[[ $(curl -sL -o /dev/null -w '%{http_code}' --max-time 5 "http://localhost:$port$path" 2>/dev/null) == 200 ]]
}

await_health() {
	local names=(web) name _rest pending deadline=$((SECONDS + health_timeout))
	while read -r name _rest; do names+=("$name"); done < <(backend_processes)
	while :; do
		pending=()
		for name in "${names[@]}"; do
			healthy "$name" && continue
			if ! pid_of "$name" >/dev/null; then
				if grep -qE 'address already in use|EADDRINUSE' "$state/$name.log"; then return "$port_conflict"; fi
				echo "make env.$command: $name exited before it was healthy; the end of $state/$name.log:" >&2
				tail -n 30 "$state/$name.log" >&2
				return 1
			fi
			pending+=("$name")
		done
		[[ ${#pending[@]} -gt 0 ]] || return 0
		if [[ $SECONDS -ge $deadline ]]; then
			echo "make env.$command: not healthy after ${health_timeout}s: ${pending[*]} (logs in $state)" >&2
			return 1
		fi
		sleep 1
	done
}

up() {
	check_o11y
	make -s -C "$root/backend" env
	make -s -C "$root/frontend" env
	stop_processes
	say "containers of project $project"
	start_containers
	rm -f "$env_file"
	read_container_ports
	allocate_process_ports
	bootstrap
	build_processes
	local attempt outcome
	for attempt in $(seq "$port_attempts"); do
		say "start processes (attempt $attempt)"
		start_processes
		outcome=0
		await_health || outcome=$?
		[[ $outcome -ne 0 ]] || break
		stop_processes
		[[ $outcome -eq $port_conflict && $attempt -lt $port_attempts ]] || fail "environment $project did not come up"
		allocate_process_ports
	done
	write_env_file
	say "environment $project is up"
	cat "$env_file"
}

down() {
	stop_processes
	say "remove project $project"
	remove_project "$project"
	rm -rf "$state" "$env_file"
}

status() {
	if [[ -f $env_file ]]; then cat "$env_file"; else echo "no $env_file: run make env.up"; fi
	echo
	local name
	for name in $(spawned_processes); do
		if pid_of "$name" >/dev/null; then echo "$name: running (pid $(cat "$state/$name.pid"))"; else echo "$name: stopped"; fi
	done
	echo
	docker compose -p "$project" ps -a
}

prune() {
	local name dir
	docker ps -a --filter "label=$ephemeral_label" \
		--format '{{.Label "com.docker.compose.project"}} {{.Label "com.docker.compose.project.working_dir"}}' |
		sort -u |
		while read -r name dir; do
			[[ -n $name && $name != backend ]] || continue
			[[ ! -d $dir ]] || continue
			say "remove project $name (its worktree $dir is gone)"
			remove_project "$name"
		done
}

case $command in
up) project=$(project_name) && up ;;
down) project=$(project_name) && down ;;
status) project=$(project_name) && status ;;
prune) prune ;;
*) fail "usage: make env.up | env.down | env.status | env.prune" ;;
esac
