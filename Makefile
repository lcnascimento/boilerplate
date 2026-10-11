# Template bootstrap. Each target deletes itself, and its script, after a
# successful run; this file goes away once both have run.

.PHONY: init
init:
	@bash scripts/init.sh "$(NAME)"

.PHONY: clean-example
clean-example:
	@bash scripts/clean-example.sh
