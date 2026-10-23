import "server-only";

import { sql, type Kysely } from "kysely";

import type { OrganizationId, UserId } from "./actor-ids";

export async function lockOrganization(
  connection: Kysely<unknown>,
  organizationId: OrganizationId
): Promise<void> {
  await sql`SELECT id FROM organization WHERE id = ${organizationId} FOR UPDATE`.execute(
    connection
  );
}

export async function lockOrganizationsOfMember(
  connection: Kysely<unknown>,
  userId: UserId
): Promise<void> {
  await sql`SELECT id FROM organization
             WHERE id IN (SELECT "organizationId" FROM member WHERE "userId" = ${userId})
             ORDER BY id
               FOR UPDATE`.execute(connection);
}
