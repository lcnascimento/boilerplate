import { defineConfig, devices } from "@playwright/test";

const CI = Boolean(process.env.CI);

// Its own port, so a `next dev` on 3000 (Tilt) is never reused by mistake.
// It must not clash with the observability stack (3010, 3100, 3200, ...).
const WEB_PORT = 3001;
const WEB_URL = `http://localhost:${WEB_PORT}`;
// example:begin
// Not Tilt's 8080: the e2e run gets its own notes service.
const NOTES_PORT = 8081;
export const NOTES_URL = `http://localhost:${NOTES_PORT}`;
// example:end

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
  projects: [
    // example:begin
    // Empties the store and asserts the empty list, so it runs before any
    // test creates a note.
    {
      name: "notes-empty",
      testMatch: "notes-empty.spec.ts",
      use: { ...devices["Desktop Chrome"] },
    },
    // example:end
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      // example:begin
      testIgnore: "notes-empty.spec.ts",
      dependencies: ["notes-empty"],
      // example:end
    },
  ],
  webServer: [
    {
      command: `next build && next start --port ${WEB_PORT}`,
      url: WEB_URL,
      reuseExistingServer: !CI,
      timeout: 180_000,
      // example:begin
      env: { NOTES_API_URL: NOTES_URL },
      // example:end
    },
    // example:begin
    {
      // Never reused, so no other run shares its store. UI Mode keeps it up
      // across runs, so the store is not empty on a rerun: notes-empty
      // empties it first.
      command: `make -s -C ../../../backend/services/notes build.api.frontend run.api.frontend FRONTEND_PORT=${NOTES_PORT}`,
      url: `${NOTES_URL}/health`,
      reuseExistingServer: false,
      timeout: 180_000,
      // e2e checks behaviour, not telemetry: no collector is needed.
      env: { OTEL_SDK_DISABLED: "true" },
    },
    // example:end
  ],
});
