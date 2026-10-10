import { defineConfig } from "vitest/config";

// Runs every package's and app's tests from the root (`pnpm vitest`). CI runs
// them per package through `turbo run test`.
export default defineConfig({
  test: {
    projects: ["packages/*/vitest.config.ts", "apps/*/vitest.config.ts"],
  },
});
