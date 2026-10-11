-include .env
export

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

GOLANGCI_LINT_VERSION := v2.14.0
MOCKGEN_VERSION := v0.6.0
GOVULNCHECK_VERSION := v1.8.0

GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

# Every Go module listed in go.work. Targets iterate them, so they work with zero or N services.
MODULES := $(shell go list -m -f '{{.Dir}}')

# Packages of the current module, without entrypoints and generated mocks.
PACKAGES = go list ./... | { grep -v -e '/cmd/' -e '/mocks$$' || true; }

# in_modules runs a shell snippet inside each module of go.work, stopping at the first failure.
define in_modules
	@for dir in $(MODULES); do \
		echo "==> $(1) $${dir#$(CURDIR)/}"; \
		(cd "$$dir" && $(2)); \
	done
endef

.PHONY: setup
setup:
	@if [ ! -f .env ]; then echo "==> Creating .env from .env.example"; cp .env.example .env; fi
	@for example in services/*/.env.example; do \
		[ -f "$$example" ] || continue; \
		env_file="$${example%.example}"; \
		if [ ! -f "$$env_file" ]; then echo "==> Creating $$env_file"; cp "$$example" "$$env_file"; fi; \
	done
	@echo "==> Installing Go tooling into $(GOBIN)"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: lint
lint:
	$(call in_modules,lint,if [ -z "$$($(PACKAGES))" ]; then echo "no packages"; else $(GOBIN)/golangci-lint run --timeout 10m ./...; fi)

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
	$(call in_modules,test,pkgs=$$($(PACKAGES)); \
		if [ -z "$$pkgs" ]; then echo "no packages"; else go test -timeout 30s -race -cover -coverprofile=coverage.out $$pkgs; fi)

.PHONY: vuln-check
vuln-check:
	$(call in_modules,vuln-check,pkgs=$$($(PACKAGES)); \
		if [ -z "$$pkgs" ]; then echo "no packages"; else $(GOBIN)/govulncheck ./...; fi)
