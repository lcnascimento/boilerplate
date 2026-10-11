import jsxA11y from "eslint-plugin-jsx-a11y";
import reactPlugin from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import { defineConfig } from "eslint/config";
import globals from "globals";

import { base } from "./base.js";

/**
 * React preset: base + React, hooks (with the React Compiler rules) and a11y.
 */
export const react = defineConfig(base, {
  files: ["**/*.tsx", "**/*.jsx"],
  extends: [
    reactPlugin.configs.flat.recommended,
    reactPlugin.configs.flat["jsx-runtime"],
    reactHooks.configs.flat.recommended,
    jsxA11y.flatConfigs.recommended,
  ],
  languageOptions: {
    globals: {
      ...globals.browser,
    },
  },
  settings: {
    react: { version: "detect" },
  },
  rules: {
    "react/prop-types": "off",
    "react/jsx-no-target-blank": "error",
    "react/jsx-curly-brace-presence": [
      "error",
      { props: "never", children: "never" },
    ],
    "react/jsx-boolean-value": ["error", "never"],
    "react/self-closing-comp": "error",
  },
});

export default react;
