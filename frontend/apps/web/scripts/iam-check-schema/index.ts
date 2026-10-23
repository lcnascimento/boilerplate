import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { z } from "zod";

const packageSchema = z.object({ name: z.string(), version: z.string() });

function installedVersion(packageName: string): string {
  let dir = path.dirname(fileURLToPath(import.meta.resolve(packageName)));
  for (;;) {
    const manifest = path.join(dir, "package.json");
    if (existsSync(manifest)) {
      const { name, version } = packageSchema.parse(
        JSON.parse(readFileSync(manifest, "utf8"))
      );
      if (name === packageName) return version;
    }
    const parent = path.dirname(dir);
    if (parent === dir) throw new Error(`no package.json for ${packageName}`);
    dir = parent;
  }
}

execFileSync(
  "pnpm",
  [
    "dlx",
    `auth@${installedVersion("better-auth")}`,
    "check",
    "schema",
    "--config",
    "auth.config.ts",
  ],
  { stdio: "inherit" }
);
