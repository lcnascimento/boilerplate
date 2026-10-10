import { fileURLToPath } from "node:url";

import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import { playwright } from "@vitest/browser-playwright";
import { defineProject } from "vitest/config";

// Every story is a test: it renders in headless Chromium, runs its `play`
// function and the a11y checks.
export default defineProject({
  plugins: [
    storybookTest({
      configDir: fileURLToPath(new URL(".storybook", import.meta.url)),
    }),
  ],
  test: {
    name: "ui",
    browser: {
      enabled: true,
      headless: true,
      provider: playwright(),
      // Desktop width: below 768px the sidebar renders as a sheet.
      viewport: { width: 1280, height: 800 },
      instances: [{ browser: "chromium" }],
    },
  },
});
