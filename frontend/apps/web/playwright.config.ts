import { defineConfig, devices } from "@playwright/test";

import { WEB_URL } from "./e2e/stack";

const CI = Boolean(process.env.CI);

// e2e runs against Tilt's stack, already up (`tilt up`): the web app and each
// backend service, at the URLs in `e2e/stack.ts`. It starts no server of its
// own; the global setup aborts the run when the stack does not answer. Tests
// share the dev data, so each one creates its own, marked `e2e-`, and the
// `marker` fixture deletes it afterwards.
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  fullyParallel: true,
  forbidOnly: CI,
  retries: CI ? 2 : 0,
  reporter: CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: WEB_URL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
