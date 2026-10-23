.PHONY: env setup lint lint.fix format format.check typecheck test test.unit test.component test.e2e build

env:
	@example=apps/web/.env.example; env_file=apps/web/.env.local; \
	if [ ! -f "$$env_file" ]; then echo "==> Creating $$env_file"; cp "$$example" "$$env_file"; exit 0; fi; \
	missing=$$(awk -F= 'NR == FNR { if (sub(/^#?[ \t]*/, "", $$1)) seen[$$1] = 1; next } /^[A-Za-z_][A-Za-z0-9_]*=/ && !($$1 in seen)' "$$env_file" "$$example"); \
	[ -n "$$missing" ] || exit 0; \
	echo "==> Adding $$(printf '%s\n' "$$missing" | cut -d= -f1 | paste -sd, -) to $$env_file"; \
	[ -z "$$(tail -c1 "$$env_file")" ] || echo >> "$$env_file"; \
	printf '%s\n' "$$missing" >> "$$env_file"

setup: env
	pnpm install --frozen-lockfile

lint:
	pnpm turbo run lint

lint.fix:
	pnpm turbo run lint:fix

format:
	pnpm format

format.check:
	pnpm format:check

typecheck:
	pnpm turbo run typecheck

test:
	pnpm turbo run test

test.unit:
	pnpm turbo run test --filter='!./packages/ui'

test.component:
	pnpm turbo run test --filter=./packages/ui

test.e2e:
	set -a && . apps/web/.env.local && set +a && \
		pnpm --filter @boilerplate/web iam:check-schema && \
		pnpm --filter @boilerplate/web exec vitest run shared/auth/actor-token.test.ts shared/auth/ownership.test.ts shared/auth/request-actor.test.ts
	pnpm turbo run test:e2e

build:
	pnpm turbo run build
