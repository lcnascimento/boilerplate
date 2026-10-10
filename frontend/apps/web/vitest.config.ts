import { defineProject } from "vitest/config";

// Unit tests run in `node`; component tests (`*.test.tsx`) opt into jsdom with
// a `@vitest-environment jsdom` docblock.
export default defineProject({
  resolve: { tsconfigPaths: true },
  test: {
    name: "web",
    environment: "node",
    include: ["**/*.test.{ts,tsx}"],
    exclude: ["**/node_modules/**", "e2e/**", ".next/**"],
  },
});
