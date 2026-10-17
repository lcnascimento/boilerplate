import { fileURLToPath } from "node:url";

import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import { playwright } from "@vitest/browser-playwright";
import { defineProject } from "vitest/config";

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
      viewport: { width: 1280, height: 800 },
      instances: [{ browser: "chromium" }],
    },
  },
});
