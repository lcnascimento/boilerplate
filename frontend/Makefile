.PHONY: setup lint lint.fix format format.check typecheck test test.unit test.component test.e2e build

setup:
	pnpm install --frozen-lockfile

lint:
	pnpm turbo run lint

lint.fix:
	pnpm turbo run lint:fix

format:
	pnpm format

format.check:
	pnpm format:check

typecheck:
	pnpm turbo run typecheck

test:
	pnpm turbo run test

test.unit:
	pnpm turbo run test --filter='!./packages/ui'

test.component:
	pnpm turbo run test --filter=./packages/ui

test.e2e:
	pnpm turbo run test:e2e

build:
	pnpm turbo run build
