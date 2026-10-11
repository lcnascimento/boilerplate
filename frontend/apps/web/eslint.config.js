import { defineConfig } from "eslint/config";

import { next } from "@boilerplate/eslint-config";

export default defineConfig(
  { ignores: ["next-env.d.ts", "playwright-report/**", "test-results/**"] },
  next
);
