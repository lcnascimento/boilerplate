AGAINST ?= origin/main
TILT_LOG ?= /tmp/tilt-up.log
TILT_MANUAL_INIT_TRIGGER_MODES := [2, 3]

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
	@$(MAKE) -C backend lint o11y.lint proto.lint
	@$(MAKE) -C frontend lint typecheck build
	@echo "==> vacuum lint openapi"
	@specs=$$(ls openapi/*.yaml 2>/dev/null); \
	if [ -z "$$specs" ]; then echo "no OpenAPI specs"; else \
		docker run --rm -v "$(CURDIR):/work:ro" -w /work dshanley/vacuum:v0.33.0 \
			lint --details --no-banner --no-style --no-update-check $$specs; \
	fi

.PHONY: lint.breaking
lint.breaking:
	@$(MAKE) -C backend proto.breaking AGAINST=$(AGAINST)

.PHONY: test
test:
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

.PHONY: test.e2e
test.e2e: test.frontend.e2e
	@$(MAKE) -C backend o11y.check-live

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
