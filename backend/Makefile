-include .env
export

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

GOLANGCI_LINT_VERSION := v2.14.0
MOCKGEN_VERSION := v0.6.0
GOVULNCHECK_VERSION := v1.8.0
GOOSE_VERSION := v3.29.0
SQUAWK_VERSION := v2.68.0
SQLC_VERSION := v1.31.1
BUF_VERSION := v1.74.0
PROTOC_GEN_GO_VERSION := v1.36.12
SLOTH_VERSION := v0.16.0
PROMTOOL_VERSION := v3.15.0
DASHBOARD_LINTER_VERSION := v0.3.0
PINT_VERSION := v0.89.3
LOKITOOL_VERSION := v3.7.8
GITLEAKS_VERSION := v8.30.1
ALLOY_IMAGE :=$(shell awk '/image: grafana\/alloy:/ {print $$2}' docker-compose.observability.yml)

GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
export PATH := $(GOBIN):$(PATH)

MODULES := $(shell go list -m -f '{{.Dir}}')

BUF_DIRS := $(patsubst %/buf.yaml,%,$(shell find services -name buf.yaml -not -path '*/node_modules/*'))
AGAINST ?= origin/main

MIGRATION_DIRS := $(wildcard services/*/internal/repository/migrations)
MIGRATIONS := $(wildcard $(addsuffix /*.sql,$(MIGRATION_DIRS)))
MIGRATIONS_WITHOUT_TRANSACTION := $(if $(MIGRATIONS),$(shell grep -l '^-- +goose NO TRANSACTION' $(MIGRATIONS)))
MIGRATIONS_IN_TRANSACTION := $(filter-out $(MIGRATIONS_WITHOUT_TRANSACTION),$(MIGRATIONS))

PACKAGES = go list ./... | { grep -v -e '/cmd/' -e '/mocks$$' || true; }

# make 3.81 (macOS) ignores .SHELLFLAGS, and `( ... ) || exit` turns errexit off inside the subshell.
define in_modules
	@for dir in $(MODULES); do \
		echo "==> $(1) $${dir#$(CURDIR)/}"; \
		(set -eu -o pipefail; cd "$$dir"; $(2)); status=$$?; \
		if [ $$status -ne 0 ]; then exit $$status; fi; \
	done
endef

define in_buf_dirs
	@if [ -z "$(BUF_DIRS)" ]; then echo "no buf.yaml"; fi
	@for dir in $(BUF_DIRS); do \
		echo "==> $(1) $$dir"; \
		(set -eu -o pipefail; cd "$$dir"; $(2)); status=$$?; \
		if [ $$status -ne 0 ]; then exit $$status; fi; \
	done
endef

define install_release
	@bin="$(GOBIN)/$(1)"; \
	if [ "$$({ $(2); } 2>/dev/null)" != "$(3)" ]; then \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		case "$$(uname -m)" in x86_64|amd64) arch=amd64;; aarch64|arm64) arch=arm64;; *) echo "$(1): unsupported arch $$(uname -m)" >&2; exit 1;; esac; \
		echo "==> Installing $(1) $(3) into $(GOBIN)"; \
		$(4) && chmod +x "$$bin"; \
	fi
endef

go_bin = $(notdir $(if $(filter v2 v3 v4 v5 v6 v7 v8 v9,$(notdir $(1))),$(patsubst %/,%,$(dir $(1))),$(1)))

define go_install
	@have=$$(go version -m "$(GOBIN)/$(call go_bin,$(1))" 2>/dev/null | awk -v build="$(3)" ' \
		BEGIN { split(build, settings, " "); for (i in settings) want[settings[i]] = 1 } \
		NR == 1 { go = $$2 } $$1 == "mod" { mod = $$3 } $$1 == "build" && $$2 in want { env = env " " $$2 } \
		END { print go, mod env }'); \
	if [ "$$have" != "$$(go env GOVERSION) $(strip $(2) $(3))" ]; then \
		echo "==> Installing $(call go_bin,$(1)) $(2) into $(GOBIN)"; \
		$(3) go install $(1)@$(2); \
	fi
endef

define await_testcontainers
	@for i in $$(seq 60); do \
		[ -n "$$(docker ps -aq --filter label=org.testcontainers)" ] || exit 0; \
		[ "$$i" -gt 1 ] || echo "==> Waiting for testcontainers to stop"; \
		sleep 1; \
	done; \
	echo "testcontainers still running after 60s:" >&2; \
	docker ps -a --filter label=org.testcontainers >&2; \
	exit 1
endef

go_test = pkgs=$$($(PACKAGES)); if [ -z "$$pkgs" ]; then echo "no packages"; else go test -timeout $(1) -race -cover $(2) $$pkgs; fi

.PHONY: env
env:
	@for example in .env.example services/*/.env.example; do \
		[ -f "$$example" ] || continue; \
		env_file="$${example%.example}"; \
		if [ ! -f "$$env_file" ]; then echo "==> Creating $$env_file"; cp "$$example" "$$env_file"; continue; fi; \
		missing=$$(awk -F= 'NR == FNR { if (sub(/^#?[ \t]*/, "", $$1)) seen[$$1] = 1; next } /^[A-Za-z_][A-Za-z0-9_]*=/ && !($$1 in seen)' "$$env_file" "$$example"); \
		[ -n "$$missing" ] || continue; \
		echo "==> Adding $$(printf '%s\n' "$$missing" | cut -d= -f1 | paste -sd, -) to $$env_file"; \
		[ -z "$$(tail -c1 "$$env_file")" ] || echo >> "$$env_file"; \
		printf '%s\n' "$$missing" >> "$$env_file"; \
	done

.PHONY: setup
setup: env
	$(call go_install,github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	$(call go_install,go.uber.org/mock/mockgen,$(MOCKGEN_VERSION))
	$(call go_install,golang.org/x/vuln/cmd/govulncheck,$(GOVULNCHECK_VERSION))
	$(call go_install,github.com/pressly/goose/v3/cmd/goose,$(GOOSE_VERSION))
	$(call go_install,github.com/sqlc-dev/sqlc/cmd/sqlc,$(SQLC_VERSION),CGO_ENABLED=1)
	@$(MAKE) -s squawk proto.tools o11y.tools gitleaks

.PHONY: gitleaks
gitleaks:
	$(call go_install,github.com/zricethezav/gitleaks/v8,$(GITLEAKS_VERSION))

.PHONY: proto.tools
proto.tools:
	$(call go_install,github.com/bufbuild/buf/cmd/buf,$(BUF_VERSION))
	$(call go_install,google.golang.org/protobuf/cmd/protoc-gen-go,$(PROTOC_GEN_GO_VERSION))

.PHONY: squawk
squawk:
	@if [ "$$($(GOBIN)/squawk --version 2>/dev/null)" != "squawk $(SQUAWK_VERSION:v%=%)" ]; then \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		case "$$(uname -m)" in x86_64|amd64) arch=x64;; aarch64|arm64) arch=arm64;; *) echo "squawk: unsupported arch $$(uname -m)" >&2; exit 1;; esac; \
		echo "==> Installing squawk $(SQUAWK_VERSION) into $(GOBIN)"; \
		curl -fsSL -o "$(GOBIN)/squawk" "https://github.com/sbdchd/squawk/releases/download/$(SQUAWK_VERSION)/squawk-$$os-$$arch" \
			&& chmod +x "$(GOBIN)/squawk"; \
	fi

.PHONY: o11y.tools
o11y.tools:
	$(call install_release,dashboard-linter,go version -m "$$bin" | grep -o 'dashboard-linter.v[0-9.]*' | cut -c18-,$(DASHBOARD_LINTER_VERSION),\
		curl -fsSL "https://github.com/grafana/dashboard-linter/releases/download/$(DASHBOARD_LINTER_VERSION)/dashboard-linter_$(DASHBOARD_LINTER_VERSION:v%=%)_$${os}_$$arch.tar.gz" \
			| tar -xzO dashboard-linter > "$$bin")
	$(call install_release,sloth,"$$bin" version,$(SLOTH_VERSION),\
		curl -fsSL -o "$$bin" "https://github.com/slok/sloth/releases/download/$(SLOTH_VERSION)/sloth-$$os-$$arch")
	$(call install_release,promtool,"$$bin" --version | head -1 | cut -d' ' -f3,$(PROMTOOL_VERSION:v%=%),\
		curl -fsSL "https://github.com/prometheus/prometheus/releases/download/$(PROMTOOL_VERSION)/prometheus-$(PROMTOOL_VERSION:v%=%).$$os-$$arch.tar.gz" \
			| tar -xzO "prometheus-$(PROMTOOL_VERSION:v%=%).$$os-$$arch/promtool" > "$$bin")
	$(call install_release,pint,"$$bin" version | cut -d' ' -f1,$(PINT_VERSION:v%=%),\
		curl -fsSL "https://github.com/cloudflare/pint/releases/download/$(PINT_VERSION)/pint-$(PINT_VERSION:v%=%)-$$os-$$arch.tar.gz" \
			| tar -xzO "pint-$$os-$$arch" > "$$bin")
	$(call install_release,lokitool,"$$bin" version | head -1 | cut -d' ' -f3,$(LOKITOOL_VERSION:v%=%),\
		zip=$$(mktemp) && curl -fsSL -o "$$zip" "https://github.com/grafana/loki/releases/download/$(LOKITOOL_VERSION)/lokitool-$$os-$$arch.zip" \
			&& unzip -p "$$zip" > "$$bin" && rm -f "$$zip")

.PHONY: lint
lint:
	$(call in_modules,lint,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint run --timeout 10m ./...; fi)
	@echo "==> lint migrations"
	@if [ -z "$(MIGRATIONS)" ]; then echo "no migrations"; fi
	@if [ -n "$(MIGRATIONS_IN_TRANSACTION)" ]; then $(GOBIN)/squawk --config .squawk.toml $(MIGRATIONS_IN_TRANSACTION); fi
	@if [ -n "$(MIGRATIONS_WITHOUT_TRANSACTION)" ]; then \
		$(GOBIN)/squawk --config .squawk.toml --no-assume-in-transaction $(MIGRATIONS_WITHOUT_TRANSACTION); fi

.PHONY: lint.fix
lint.fix:
	$(call in_modules,lint.fix,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint run --fix --timeout 10m ./...; fi)

.PHONY: format
format:
	$(call in_modules,format,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint fmt ./...; fi)

.PHONY: lint.cache.clean
lint.cache.clean:
	@$(GOBIN)/golangci-lint cache clean

.PHONY: tidy
tidy:
	$(call in_modules,tidy,go mod tidy)

.PHONY: mocks
mocks:
	$(call in_modules,mocks,find . -path '*/internal/mocks/mocks.go' -delete; \
		for src in $$(find . -name contracts.go); do \
			dir=$$(dirname "$$src"); \
			$(GOBIN)/mockgen -source "$$src" -destination "$$dir/internal/mocks/mocks.go" -package mocks; \
		done)

.PHONY: postgres
postgres:
	@docker compose up -d --wait postgres

.PHONY: test
test: postgres
	$(call in_modules,test,$(call go_test,3m,-coverprofile=coverage.out))
	$(await_testcontainers)

.PHONY: test.unit
test.unit:
	$(call in_modules,test.unit,$(call go_test,30s,-short))

.PHONY: test.component
test.component: postgres
	$(call in_modules,test.component,$(call go_test,3m,-run Component))
	$(await_testcontainers)

.PHONY: sqlc
sqlc:
	$(call in_modules,sqlc,if [ -f sqlc.yaml ]; then $(GOBIN)/sqlc generate && $(GOBIN)/sqlc vet; else echo "no sqlc.yaml"; fi)

.PHONY: o11y.generate
o11y.generate:
	@cd tools && GOWORK=off go run ./o11ygen -root ..

.PHONY: iam.generate
iam.generate:
	@cd tools && GOWORK=off go run ./iamgen -out ../../frontend/apps/web/shared/auth/authz.gen.ts

.PHONY: o11y.lint
o11y.lint: o11y.lint.dashboards o11y.lint.rules o11y.lint.loki o11y.lint.slo o11y.lint.alloy

.PHONY: o11y.lint.rules
o11y.lint.rules:
	@echo "==> promtool check rules"
	@$(GOBIN)/promtool check rules config/prometheus/rules/*.yaml config/prometheus/tests/fixture*.yaml
	@echo "==> promtool test rules"
	@$(GOBIN)/promtool test rules config/prometheus/tests/*.test.yaml
	@echo "==> pint lint"
	@$(GOBIN)/pint --no-color --log-level warn --offline lint config/prometheus/rules config/prometheus/tests/fixture*.yaml

.PHONY: o11y.lint.slo
o11y.lint.slo:
	@if ! ls config/sloth/*.yaml >/dev/null 2>&1; then echo "no SLO specs"; exit 0; fi; \
	echo "==> sloth validate" && $(GOBIN)/sloth validate --no-log --input config/sloth \
	&& echo "==> promtool check rules (SLO)" && $(GOBIN)/promtool check rules config/prometheus/slo/*.yaml \
	&& echo "==> pint lint (SLO)" \
	&& $(GOBIN)/pint --no-color --log-level warn --offline --disabled rule/dependency lint config/prometheus/slo

.PHONY: o11y.lint.dashboards
o11y.lint.dashboards:
	@for dashboard in config/grafana/provisioning/dashboards/*.json; do \
		echo "==> dashboard-linter $$dashboard"; \
		out=$$($(GOBIN)/dashboard-linter lint --strict "$$dashboard" 2>&1) || { echo "$$out" | grep -v '^Checks that'; exit 1; }; \
	done

.PHONY: o11y.lint.loki
o11y.lint.loki:
	@rules=$$(find config/loki/rules -name '*.yaml' | sort); \
	if [ -z "$$rules" ]; then echo "==> lokitool: no Loki rules"; exit 0; fi; \
	echo "==> lokitool rules lint"; \
	out=$$($(GOBIN)/lokitool rules lint --dry-run $$rules 2>&1) || { echo "$$out"; exit 1; }; \
	echo "==> lokitool rules check"; \
	$(GOBIN)/lokitool rules check --strict $$rules

.PHONY: o11y.lint.alloy
o11y.lint.alloy:
	@echo "==> alloy validate"
	@docker run --rm -i --entrypoint sh $(ALLOY_IMAGE) -c 'cat > /tmp/config.alloy && alloy validate /tmp/config.alloy' < config/alloy/config.alloy

.PHONY: o11y.check-live
o11y.check-live:
	@cd tools && \
		O11Y_LIVE_PROMETHEUS_URL="$${O11Y_LIVE_PROMETHEUS_URL:-http://localhost:9090}" \
		O11Y_LIVE_LOKI_URL="$${O11Y_LIVE_LOKI_URL:-http://localhost:3100}" \
		O11Y_LIVE_GRAFANA_URL="$${O11Y_LIVE_GRAFANA_URL:-http://admin:admin@localhost:3010}" \
		go test -count=1 -timeout 6m -v -run '^TestComponentStack_ServesGeneratedArtifacts$$' ./o11ygen

.PHONY: grafana.token
grafana.token:
	@bash scripts/grafana-token.sh

.PHONY: proto.lint
proto.lint:
	$(call in_buf_dirs,proto.lint,$(GOBIN)/buf lint)

.PHONY: proto.generate
proto.generate:
	$(call in_buf_dirs,proto.generate,find . -maxdepth 1 -name '*.pb.go' -delete; PATH="$(GOBIN):$$PATH" $(GOBIN)/buf generate)

.PHONY: proto.breaking
proto.breaking:
	$(call in_buf_dirs,proto.breaking,module=$$(git rev-parse --show-prefix); \
		if ! git cat-file -e "$(AGAINST):$${module}buf.yaml" 2>/dev/null; then echo "$(AGAINST) has no buf module at $$module"; exit 0; fi; \
		against=$$(mktemp -d); trap 'rm -rf "$$against"' EXIT; \
		git -C "$$(git rev-parse --show-toplevel)" archive "$(AGAINST)" "$$module" | tar -x -C "$$against"; \
		$(GOBIN)/buf breaking --against "$$against/$$module")

.PHONY: vuln-check
vuln-check:
	$(call in_modules,vuln-check,pkgs=$$($(PACKAGES)); \
		if [ -z "$$pkgs" ]; then echo "no packages"; else $(GOBIN)/govulncheck ./...; fi)

.PHONY: migration.new
migration.new:
	@[ -n "$(name)" ] || { echo "usage: make migration.new name=<snake_case> [service=<name>]" >&2; exit 1; }
	@set -- $(if $(service),services/$(service)/internal/repository/migrations,$(MIGRATION_DIRS)); \
	if [ $$# -ne 1 ] || [ ! -d "$$1" ]; then \
		echo "make migration.new: pick a service with service=<name> (services with migrations: $(or $(MIGRATION_DIRS:services/%/internal/repository/migrations=%),none))" >&2; \
		exit 1; \
	fi; \
	file=$$($(GOBIN)/goose -dir "$$1" -s create "$(name)" sql 2>&1 | sed -n 's/.*Created new file: //p'); \
	[ -n "$$file" ] || { echo "make migration.new: goose did not create the migration" >&2; exit 1; }; \
	printf -- '-- +goose Up\n\n-- +goose Down\n' > "$$file"; \
	echo "==> Created $$file"
