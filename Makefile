.PHONY: init
init:
	@bash scripts/init.sh "$(NAME)"

.PHONY: clean-example
clean-example:
	@bash scripts/clean-example.sh

.PHONY: grafana.token
grafana.token:
	@bash scripts/grafana-token.sh
