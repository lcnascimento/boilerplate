import storybook from "eslint-plugin-storybook";
import { defineConfig } from "eslint/config";

import { react } from "@boilerplate/eslint-config";

export default defineConfig(react, storybook.configs["flat/recommended"]);
