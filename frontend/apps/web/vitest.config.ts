import { defineProject } from "vitest/config";

export default defineProject({
  test: {
    name: "web",
    environment: "node",
    include: ["**/*.test.ts"],
    exclude: ["**/node_modules/**", "e2e/**", ".next/**"],
  },
});
