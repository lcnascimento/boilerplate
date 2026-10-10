import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import { mergeConfig } from "vite";

import type { StorybookConfig } from "@storybook/react-vite";

/** Absolute path of a package, as Storybook needs in a pnpm monorepo. */
function packagePath(name: string): string {
  return dirname(fileURLToPath(import.meta.resolve(`${name}/package.json`)));
}

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.tsx"],
  addons: [
    packagePath("@storybook/addon-docs"),
    packagePath("@storybook/addon-a11y"),
    packagePath("@storybook/addon-vitest"),
  ],
  framework: packagePath("@storybook/react-vite"),
  viteFinal: (viteConfig) =>
    mergeConfig(viteConfig, {
      plugins: [tailwindcss()],
      // Components import each other as `@boilerplate/ui/*` (shadcn aliases),
      // mapped to `src/*` by tsconfig `paths`.
      resolve: { tsconfigPaths: true },
    }),
};

export default config;
