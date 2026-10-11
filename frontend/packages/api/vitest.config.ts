import { defineProject } from "vitest/config";

export default defineProject({
  // `server-only` throws outside a React Server Components bundle. Inline it
  // and resolve with the `react-server` condition, as Next does on the server,
  // so it resolves to its empty module.
  ssr: { resolve: { conditions: ["react-server"] } },
  test: {
    name: "api",
    environment: "node",
    include: ["src/**/*.test.ts"],
    server: { deps: { inline: ["server-only"] } },
  },
});
