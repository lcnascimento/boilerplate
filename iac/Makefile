SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

TOFU_VERSION := v1.13.2

GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)
TOFU := $(GOBIN)/tofu

# make 3.81 (macOS) ignores .SHELLFLAGS, so recipes that prompt set their own flags.
define open_state
	set -euo pipefail; \
	read -rsp "State passphrase: " passphrase; echo >&2; \
	previous=; \
	if grep -q '^variable "previous_state_passphrase"' *.tf; then read -rsp "Previous state passphrase: " previous; echo >&2; fi; \
	tofu() { TF_VAR_state_passphrase="$$passphrase" TF_VAR_previous_state_passphrase="$$previous" DOPPLER_TOKEN="$${token-}" "$(TOFU)" "$$@"; }; \
	tofu init -input=false >/dev/null
endef

.PHONY: tools
tools:
	@if [ "$$($(TOFU) --version 2>/dev/null | head -1)" != "OpenTofu $(TOFU_VERSION)" ]; then \
		os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		case "$$(uname -m)" in x86_64|amd64) arch=amd64;; aarch64|arm64) arch=arm64;; *) echo "tofu: unsupported arch $$(uname -m)" >&2; exit 1;; esac; \
		echo "==> Installing tofu $(TOFU_VERSION) into $(GOBIN)"; \
		mkdir -p "$(GOBIN)"; \
		curl -fsSL "https://github.com/opentofu/opentofu/releases/download/$(TOFU_VERSION)/tofu_$(TOFU_VERSION:v%=%)_$${os}_$${arch}.tar.gz" \
			| tar -xzO tofu > "$(TOFU)" && chmod +x "$(TOFU)"; \
	fi

.PHONY: format
format: tools
	@$(TOFU) fmt -recursive

.PHONY: check
check: tools
	@$(TOFU) fmt -check -recursive
	@$(TOFU) init -backend=false -input=false >/dev/null
	@$(TOFU) validate

.PHONY: plan apply
plan apply: tools
	@token=$$(doppler configure get token --plain); \
	[ -n "$$token" ] || { echo "no Doppler token: run doppler login first" >&2; exit 1; }; \
	$(call open_state); \
	tofu $@ $(if $(REPLACE),-replace='$(REPLACE)')

.PHONY: dev-token
dev-token: tools
	@[ -n "$(NAME)" ] || { echo "NAME is required (make iac.dev-token NAME=<holder>)" >&2; exit 1; }
	@$(call open_state); \
	tofu output -json dev_tokens | jq -er --arg name "$(NAME)" '.[$$name] // error("no dev token for \($$name)")'
