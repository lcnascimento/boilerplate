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

.PHONY: test
test:
	@$(MAKE) -C backend test
	@$(MAKE) -C frontend test
