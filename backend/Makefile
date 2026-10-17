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

GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

# Every Go module listed in go.work. Targets iterate them, so they work with zero or N services.
MODULES := $(shell go list -m -f '{{.Dir}}')

MIGRATION_DIRS := $(wildcard services/*/internal/repository/migrations)
MIGRATIONS := $(wildcard $(addsuffix /*.sql,$(MIGRATION_DIRS)))

# Packages of the current module, without entrypoints and generated mocks.
PACKAGES = go list ./... | { grep -v -e '/cmd/' -e '/mocks$$' || true; }

# in_modules runs a shell snippet inside each module of go.work, stopping at the first failure.
# The snippet sets its own errexit: make 3.81 (macOS) ignores .SHELLFLAGS, and the
# subshell's status is read on its own line because `( ... ) || exit` would turn errexit off inside it.
define in_modules
	@for dir in $(MODULES); do \
		echo "==> $(1) $${dir#$(CURDIR)/}"; \
		(set -eu -o pipefail; cd "$$dir"; $(2)); status=$$?; \
		if [ $$status -ne 0 ]; then exit $$status; fi; \
	done
endef

go_test = pkgs=$$($(PACKAGES)); if [ -z "$$pkgs" ]; then echo "no packages"; else go test -timeout 30s -race -cover $(1) $$pkgs; fi

.PHONY: env
env:
	@for example in .env.example services/*/.env.example; do \
		[ -f "$$example" ] || continue; \
		env_file="$${example%.example}"; \
		if [ ! -f "$$env_file" ]; then echo "==> Creating $$env_file"; cp "$$example" "$$env_file"; fi; \
	done

.PHONY: setup
setup: env
	@echo "==> Installing Go tooling into $(GOBIN)"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)
	@CGO_ENABLED=1 go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	@$(MAKE) -s squawk

.PHONY: squawk
squawk:
	@if [ "$$($(GOBIN)/squawk --version 2>/dev/null)" != "squawk $(SQUAWK_VERSION:v%=%)" ]; then \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		case "$$(uname -m)" in x86_64|amd64) arch=x64;; aarch64|arm64) arch=arm64;; *) echo "squawk: unsupported arch $$(uname -m)" >&2; exit 1;; esac; \
		echo "==> Installing squawk $(SQUAWK_VERSION) into $(GOBIN)"; \
		curl -fsSL -o "$(GOBIN)/squawk" "https://github.com/sbdchd/squawk/releases/download/$(SQUAWK_VERSION)/squawk-$$os-$$arch" \
			&& chmod +x "$(GOBIN)/squawk"; \
	fi

.PHONY: lint
lint:
	$(call in_modules,lint,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint run --timeout 10m ./...; fi)
	@echo "==> lint migrations"
	@if [ -z "$(MIGRATIONS)" ]; then echo "no migrations"; else $(GOBIN)/squawk --config .squawk.toml $(MIGRATIONS); fi

.PHONY: lint.fix
lint.fix:
	$(call in_modules,lint.fix,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint run --fix --timeout 10m ./...; fi)

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

.PHONY: test
test:
	$(call in_modules,test,$(call go_test,-coverprofile=coverage.out))

.PHONY: test.unit
test.unit:
	$(call in_modules,test.unit,$(call go_test,-short))

.PHONY: test.component
test.component:
	$(call in_modules,test.component,$(call go_test,-run Component))

.PHONY: sqlc
sqlc:
	$(call in_modules,sqlc,if [ -f sqlc.yaml ]; then $(GOBIN)/sqlc generate && $(GOBIN)/sqlc vet; else echo "no sqlc.yaml"; fi)

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
