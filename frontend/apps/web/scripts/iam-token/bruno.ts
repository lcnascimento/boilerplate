import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

import { withEnvValue } from "./dotenv";

export const BRUNO_COLLECTIONS = path.resolve(
  import.meta.dirname,
  "../../../../../backend/config/bruno/collections"
);

export const IAM_TOKEN_AUTH = `request:
  auth:
    type: bearer
    token: "{{process.env.IAM_TOKEN}}"
`;

export function brunoCollections(): string[] {
  if (!existsSync(BRUNO_COLLECTIONS)) return [];
  return readdirSync(BRUNO_COLLECTIONS, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => path.join(BRUNO_COLLECTIONS, entry.name));
}

export function writeIamToken(collection: string, token: string): string {
  const envFile = path.join(collection, ".env");
  const current = existsSync(envFile) ? readFileSync(envFile, "utf8") : "";
  writeFileSync(envFile, withEnvValue(current, "IAM_TOKEN", token));
  return envFile;
}
