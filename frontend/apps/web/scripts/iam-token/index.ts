import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { parseArgs } from "node:util";

import { mintActorToken } from "@/shared/auth/actor-token";
import { getIam } from "@/shared/auth/auth";

import { withEnvValue } from "./dotenv";
import { parseTtl } from "./ttl";

const USAGE =
  "usage: pnpm --filter @boilerplate/web iam:token --as <email> [--org <slug>] [--ttl 1h]";

const BRUNO_COLLECTIONS = path.resolve(
  import.meta.dirname,
  "../../../../../backend/config/bruno/collections"
);

const { values } = parseArgs({
  options: {
    as: { type: "string" },
    org: { type: "string" },
    ttl: { type: "string" },
  },
});

if (values.as === undefined) {
  console.error(USAGE);
  process.exit(2);
}

const { auth, database } = getIam();

async function findOrganizationId(
  userId: string,
  slug: string | undefined
): Promise<string> {
  const { adapter } = await auth.$context;
  if (slug !== undefined) {
    const organization = await adapter.findOne<{ id: string }>({
      model: "organization",
      where: [{ field: "slug", value: slug }],
    });
    if (organization === null) throw new Error(`no organization "${slug}"`);
    return organization.id;
  }

  const [oldest] = await adapter.findMany<{ organizationId: string }>({
    model: "member",
    where: [{ field: "userId", value: userId }],
    sortBy: { field: "createdAt", direction: "asc" },
    limit: 1,
  });
  if (oldest === undefined) {
    throw new Error(`${String(values.as)} is not a member of any organization`);
  }
  return oldest.organizationId;
}

function writeToBrunoCollections(token: string): string[] {
  if (!existsSync(BRUNO_COLLECTIONS)) return [];
  return readdirSync(BRUNO_COLLECTIONS, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => {
      const envFile = path.join(BRUNO_COLLECTIONS, entry.name, ".env");
      const current = existsSync(envFile) ? readFileSync(envFile, "utf8") : "";
      writeFileSync(envFile, withEnvValue(current, "IAM_TOKEN", token));
      return path.relative(process.cwd(), envFile);
    });
}

try {
  const context = await auth.$context;
  const found = await context.internalAdapter.findUserByEmail(values.as);
  if (found === null) throw new Error(`no user with email ${values.as}`);

  const token = await mintActorToken({
    userId: found.user.id,
    organizationId: await findOrganizationId(found.user.id, values.org),
    ttlSeconds: values.ttl === undefined ? undefined : parseTtl(values.ttl),
  });

  const written = writeToBrunoCollections(token);
  if (written.length === 0) console.log(token);
  for (const file of written) console.log(`wrote IAM_TOKEN to ${file}`);
} finally {
  await database.end();
}
