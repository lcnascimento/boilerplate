import "server-only";

import { sql, type Kysely } from "kysely";

import { getIam } from "./auth";
import { lockOrganization } from "./organization-lock";
import { canManageOrganization, type DeletionResult } from "./organizations";

import type { Actor } from "./actor";
import type { MemberId, OrganizationId } from "./actor-ids";

interface OrganizationOwner {
  memberId: MemberId;
  organizationId: OrganizationId;
}

type OwnedOrganizationDeletion = "deleted" | "notOwner" | "notMatched";

export async function deleteOwnedOrganization(
  connection: Kysely<unknown>,
  { memberId, organizationId }: OrganizationOwner,
  confirmation?: { name: string }
): Promise<OwnedOrganizationDeletion> {
  await lockOrganization(connection, organizationId);

  const { rows: owners } = await sql`
    SELECT 1 FROM member
     WHERE id = ${memberId} AND "organizationId" = ${organizationId} AND role = 'owner'`.execute(
    connection
  );
  if (owners.length === 0) return "notOwner";

  const nameMatches =
    confirmation === undefined ? sql`true` : sql`name = ${confirmation.name}`;
  const { numAffectedRows } = await sql`
    DELETE FROM organization WHERE id = ${organizationId} AND ${nameMatches}`.execute(
    connection
  );
  return numAffectedRows === 1n ? "deleted" : "notMatched";
}

export async function deleteOrganization(
  actor: Actor,
  confirmation: string
): Promise<DeletionResult> {
  if (!canManageOrganization(actor.role, "delete")) {
    return { kind: "error", code: "forbidden" };
  }

  const deletion = await getIam()
    .database.transaction()
    .execute((trx) =>
      deleteOwnedOrganization(trx, actor, { name: confirmation })
    );
  switch (deletion) {
    case "deleted":
      return { kind: "ok" };
    case "notOwner":
      return { kind: "error", code: "forbidden" };
    case "notMatched":
      return { kind: "error", code: "nameMismatch" };
    default:
      return deletion satisfies never;
  }
}
