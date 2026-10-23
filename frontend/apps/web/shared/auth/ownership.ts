import "server-only";

import { sql } from "kysely";

import { getIam } from "./auth";
import { assignableRoles } from "./members";
import { lockOrganization } from "./organization-lock";
import { canManageOrganization } from "./organizations";

import type { Actor } from "./actor";
import type { MemberId } from "./actor-ids";

export type OwnershipErrorCode = "invalidInput" | "forbidden" | "unexpected";

export type OwnershipResult =
  { kind: "ok" } | { kind: "error"; code: OwnershipErrorCode };

class TransferRefused extends Error {
  constructor(readonly code: OwnershipErrorCode) {
    super(`ownership transfer refused: ${code}`);
  }
}

export async function transferOwnershipFor(
  actor: Actor,
  memberId: MemberId
): Promise<OwnershipResult> {
  if (!canManageOrganization(actor.role, "transfer")) {
    return { kind: "error", code: "forbidden" };
  }

  try {
    await getIam()
      .database.transaction()
      .execute(async (trx) => {
        await lockOrganization(trx, actor.organizationId);

        const promoted = await sql`
          UPDATE member SET role = 'owner'
           WHERE id = ${memberId} AND "organizationId" = ${actor.organizationId}
             AND role = ANY(${assignableRoles}::text[])`.execute(trx);
        if (promoted.numAffectedRows !== 1n) {
          throw new TransferRefused("invalidInput");
        }

        const demoted = await sql`
          UPDATE member SET role = 'admin'
           WHERE id = ${actor.memberId} AND "organizationId" = ${actor.organizationId}
             AND role = 'owner'`.execute(trx);
        if (demoted.numAffectedRows !== 1n) {
          throw new TransferRefused("forbidden");
        }
      });
  } catch (error) {
    if (error instanceof TransferRefused) {
      return { kind: "error", code: error.code };
    }
    throw error;
  }

  return { kind: "ok" };
}
