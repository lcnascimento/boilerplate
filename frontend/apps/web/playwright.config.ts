import { defineConfig, devices } from "@playwright/test";

const CI = Boolean(process.env.CI);

// Its own port, so a `next dev` on 3000 (Tilt) is never reused by mistake.
// It must not clash with the observability stack (3010, 3100, 3200, ...).
const WEB_PORT = 3001;
const WEB_URL = `http://localhost:${WEB_PORT}`;

// e2e runs against the real stack: a production build of the web app and,
// for each service, its real binary. Every test creates its own data.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: CI,
  retries: CI ? 2 : 0,
  reporter: CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: WEB_URL,
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: [
    {
      command: `next build && next start --port ${WEB_PORT}`,
      url: WEB_URL,
      reuseExistingServer: !CI,
      timeout: 180_000,
    },
  ],
});
