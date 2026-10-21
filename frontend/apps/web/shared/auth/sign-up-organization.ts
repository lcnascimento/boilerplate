import { organizationSlug } from "./organization-slug";

import type { Pool, PoolClient } from "pg";

export type CreateOrganization = (body: {
  name: string;
  slug: string;
  userId: string;
}) => Promise<{ id: string }>;

async function pendingOrganizationName(
  db: Pool | PoolClient,
  userId: string
): Promise<string | null> {
  const { rows } = await db.query<{ pendingOrganizationName: string | null }>(
    `SELECT "pendingOrganizationName" FROM "user" WHERE id = $1`,
    [userId]
  );
  return rows[0]?.pendingOrganizationName ?? null;
}

async function hasMembership(
  client: PoolClient,
  userId: string
): Promise<boolean> {
  const { rowCount } = await client.query(
    `SELECT 1 FROM member WHERE "userId" = $1 LIMIT 1`,
    [userId]
  );
  return (rowCount ?? 0) > 0;
}

export async function ensureSignUpOrganization(
  database: Pool,
  createOrganization: CreateOrganization,
  userId: string
): Promise<void> {
  if ((await pendingOrganizationName(database, userId)) === null) return;

  const client = await database.connect();
  try {
    await client.query("BEGIN");
    await client.query(
      "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))",
      [userId]
    );

    const name = await pendingOrganizationName(client, userId);
    if (name !== null) {
      if (!(await hasMembership(client, userId))) {
        const { id } = await createOrganization({
          name,
          slug: organizationSlug(name),
          userId,
        });
        await client.query(
          `UPDATE session SET "activeOrganizationId" = $2 WHERE "userId" = $1 AND "activeOrganizationId" IS NULL`,
          [userId, id]
        );
      }
      await client.query(
        `UPDATE "user" SET "pendingOrganizationName" = NULL WHERE id = $1`,
        [userId]
      );
    }

    await client.query("COMMIT");
  } catch (error) {
    await client.query("ROLLBACK");
    throw error;
  } finally {
    client.release();
  }
}
