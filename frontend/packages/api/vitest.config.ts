import { defineProject } from "vitest/config";

export default defineProject({
  // `server-only` throws unless resolved with the `react-server` condition, as Next does.
  ssr: { resolve: { conditions: ["react-server"] } },
  test: {
    name: "api",
    environment: "node",
    include: ["src/**/*.test.ts"],
    server: { deps: { inline: ["server-only"] } },
  },
});
