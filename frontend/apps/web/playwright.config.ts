import { existsSync } from "node:fs";
import path from "node:path";

import { defineConfig, devices } from "@playwright/test";

import "./e2e/server-only";
import { storageStateOf } from "./e2e/auth-state";
import { WEB_URL } from "./e2e/stack";
import { ACME_OWNER } from "./scripts/iam-seed/users";

const ENV_LOCAL = path.join(import.meta.dirname, ".env.local");
if (existsSync(ENV_LOCAL)) process.loadEnvFile(ENV_LOCAL);

const CI = Boolean(process.env.CI);

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
  projects: [
    { name: "setup", testMatch: /auth\.setup\.ts$/ },
    {
      name: "chromium",
      dependencies: ["setup"],
      use: {
        ...devices["Desktop Chrome"],
        storageState: storageStateOf(ACME_OWNER.email),
      },
    },
  ],
});
