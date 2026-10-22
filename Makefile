.PHONY: init
init:
	@bash scripts/init.sh "$(NAME)"

.PHONY: clean-example
clean-example:
	@bash scripts/clean-example.sh

.PHONY: format
format:
	@$(MAKE) -C backend format sqlc mocks o11y.generate proto.generate
	@$(MAKE) -C frontend format

.PHONY: lint
lint:
	@$(MAKE) -C backend lint o11y.lint proto.lint proto.breaking AGAINST=origin/main
	@$(MAKE) -C frontend lint typecheck build
	@echo "==> vacuum lint openapi"
	@specs=$$(ls openapi/*.yaml 2>/dev/null); \
	if [ -z "$$specs" ]; then echo "no OpenAPI specs"; else \
		docker run --rm -v "$(CURDIR):/work:ro" -w /work dshanley/vacuum:v0.33.0 \
			lint --details --no-banner --no-style --no-update-check $$specs; \
	fi

.PHONY: infra.up
infra.up:
	@$(MAKE) -C backend infra.up

.PHONY: infra.down
infra.down:
	@$(MAKE) -C backend infra.down

.PHONY: test
test: infra.up
	@$(MAKE) -C backend test
	@$(MAKE) -C frontend test

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
