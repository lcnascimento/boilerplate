import "server-only";

import { getIam } from "./auth";
import {
  canManageOrganization,
  type DeletionErrorCode,
  type DeletionResult,
} from "./organizations";

import type { Actor } from "./actor";

class DeletionRefused extends Error {
  constructor(readonly code: DeletionErrorCode) {
    super(`organization deletion refused: ${code}`);
  }
}

export async function deleteOrganization(
  actor: Actor,
  confirmation: string
): Promise<DeletionResult> {
  if (!canManageOrganization(actor.role, "delete")) {
    return { kind: "error", code: "forbidden" };
  }

  const { adapter } = await getIam().auth.$context;
  try {
    await adapter.transaction(async (trx) => {
      const owner = await trx.findOne({
        model: "member",
        where: [
          { field: "id", value: actor.memberId },
          { field: "organizationId", value: actor.organizationId },
          { field: "role", value: "owner" },
        ],
      });
      if (owner === null) throw new DeletionRefused("forbidden");

      const deleted = await trx.deleteMany({
        model: "organization",
        where: [
          { field: "id", value: actor.organizationId },
          { field: "name", value: confirmation },
        ],
      });
      if (deleted !== 1) throw new DeletionRefused("nameMismatch");
    });
  } catch (error) {
    if (error instanceof DeletionRefused) {
      return { kind: "error", code: error.code };
    }
    throw error;
  }

  return { kind: "ok" };
}
