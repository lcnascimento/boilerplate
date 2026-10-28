TILT_LOG ?= /tmp/tilt-up.log
TILT_MANUAL_INIT_TRIGGER_MODES := [2, 3]

.PHONY: init
init:
	@bash scripts/init.sh "$(NAME)"

.PHONY: clean-example
clean-example:
	@bash scripts/clean-example.sh

.PHONY: env
env:
	@[ -f .secrets ] || { echo "==> Creating .secrets"; cp .secrets.example .secrets; }

.PHONY: setup
setup: env
	pnpm install --frozen-lockfile
	@$(MAKE) -C backend setup
	@$(MAKE) -C frontend setup
	@$(MAKE) iac.tools

.PHONY: format
format:
	@$(MAKE) -C backend format sqlc mocks o11y.generate proto.generate iam.generate tidy
	@$(MAKE) -C frontend format
	@$(MAKE) iac.format

.PHONY: format.check
format.check:
	@dirty=$$(git status --porcelain); \
	if [ -n "$$dirty" ]; then \
		echo "format.check needs a clean tree (commit, or git stash -u):"; echo "$$dirty"; exit 1; \
	fi
	@$(MAKE) format
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "make format changed the tree; commit the result:"; \
		git --no-pager diff --stat; git --no-pager diff; \
		untracked=$$(git ls-files --others --exclude-standard); \
		if [ -n "$$untracked" ]; then echo "untracked files:"; echo "$$untracked"; fi; \
		exit 1; \
	fi

.PHONY: lint
lint:
	@$(MAKE) -C backend lint o11y.lint proto.lint
	@$(MAKE) -C frontend lint typecheck build
	@echo "==> vacuum lint openapi"
	@specs=$$(ls openapi/*.yaml 2>/dev/null); \
	if [ -z "$$specs" ]; then echo "no OpenAPI specs"; else \
		docker run --rm -v "$(CURDIR):/work:ro" -w /work dshanley/vacuum:v0.33.0 \
			lint --details --no-banner --no-style --no-update-check $$specs; \
	fi

.PHONY: secrets.check
secrets.check:
	@"$(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)/gitleaks" git --no-banner --redact --verbose $(GITLEAKS_ARGS)

.PHONY: lint.breaking
lint.breaking:
	@$(MAKE) -C backend proto.breaking

.PHONY: test
test: hooks.test
	@$(MAKE) -C backend test
	@$(MAKE) -C frontend test

HOOKS_TESTS := \
	.claude/hooks/guard-secrets_test.sh \
	scripts/with-secrets_test.sh

.PHONY: hooks.test
hooks.test:
	@for t in $(HOOKS_TESTS); do bash "$$t" || exit 1; done

.PHONY: test.unit
test.unit: test.backend.unit test.frontend.unit

.PHONY: test.component
test.component: test.backend.component test.frontend.component

.PHONY: test.backend.unit
test.backend.unit:
	@$(MAKE) -C backend test.unit

.PHONY: test.backend.component
test.backend.component:
	@$(MAKE) -C backend test.component

.PHONY: test.frontend.unit
test.frontend.unit:
	@$(MAKE) -C frontend test.unit

.PHONY: test.frontend.component
test.frontend.component:
	@$(MAKE) -C frontend test.component

.PHONY: test.frontend.e2e
test.frontend.e2e:
	@$(MAKE) -C frontend test.e2e

.PHONY: test.e2e
test.e2e: test.frontend.e2e
	@$(MAKE) -C backend o11y.check-live

.PHONY: iac.tools iac.format iac.check iac.plan iac.apply iac.dev-token
iac.tools iac.format iac.check iac.plan iac.apply iac.dev-token:
	@$(MAKE) -s -C iac $(@:iac.%=%)

.PHONY: env.up env.down env.status env.prune
env.up env.down env.status env.prune:
	@bash scripts/env.sh $(@:env.%=%)

.PHONY: env.check
env.check:
	@bash scripts/env-check.sh

.PHONY: env.check.test
env.check.test:
	@bash scripts/env-check_test.sh

.PHONY: stack.up
stack.up:
	@tilt up --stream < /dev/null > "$(TILT_LOG)" 2>&1 &
	@for _ in $$(seq 60); do tilt get uiresource > /dev/null 2>&1 && break; sleep 1; done
	@tilt wait --for=condition=Ready 'uiresource/(Tiltfile)' --timeout=2m
	@tilt wait --for=condition=Ready --timeout=5m $$(tilt get uiresource -o json | \
		jq -r --argjson manual_init '$(TILT_MANUAL_INIT_TRIGGER_MODES)' \
			'.items[] | select(.status.triggerMode // 0 | IN($$manual_init[]) | not) | "uiresource/" + .metadata.name')

.PHONY: stack.down
stack.down:
	@tilt down || true
