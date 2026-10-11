import nextPlugin from "@next/eslint-plugin-next";
import { defineConfig } from "eslint/config";

import { react } from "./react.js";

/**
 * Next.js preset: react + the `@next/next` recommended and core-web-vitals rules.
 */
export const next = defineConfig(react, {
  files: ["**/*.{js,jsx,ts,tsx}"],
  plugins: {
    "@next/next": nextPlugin,
  },
  rules: {
    ...nextPlugin.configs.recommended.rules,
    ...nextPlugin.configs["core-web-vitals"].rules,
  },
});

export default next;
