import path from "node:path";
import { parseArgs } from "node:util";

import { parseActorIds } from "@/shared/auth/actor-ids";
import { mintActorToken } from "@/shared/auth/actor-token";
import { getIam } from "@/shared/auth/auth";

import { brunoCollections, writeIamToken } from "./bruno";
import { parseTtl } from "./ttl";

const USAGE =
  "usage: pnpm --filter @boilerplate/web iam:token --as <email> [--org <slug>] [--ttl 1h]";

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

try {
  const context = await auth.$context;
  const found = await context.internalAdapter.findUserByEmail(values.as);
  if (found === null) throw new Error(`no user with email ${values.as}`);

  const token = await mintActorToken({
    ...parseActorIds({
      userId: found.user.id,
      organizationId: await findOrganizationId(found.user.id, values.org),
    }),
    ttlSeconds: values.ttl === undefined ? undefined : parseTtl(values.ttl),
  });

  const collections = brunoCollections();
  if (collections.length === 0) {
    console.error("no Bruno collection to write IAM_TOKEN to; the token:");
    console.log(token);
  }
  for (const collection of collections) {
    const envFile = writeIamToken(collection, token);
    console.log(`wrote IAM_TOKEN to ${path.relative(process.cwd(), envFile)}`);
  }
} finally {
  await database.end();
}
