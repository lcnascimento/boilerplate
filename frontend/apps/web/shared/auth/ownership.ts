import "server-only";

import { getIam } from "./auth";
import { canTransferOwnership } from "./members";

import type { Actor } from "./actor";

export type OwnershipErrorCode = "invalidInput" | "forbidden" | "unexpected";

export type OwnershipResult =
  { kind: "ok" } | { kind: "error"; code: OwnershipErrorCode };

class TransferRefused extends Error {
  constructor(readonly code: OwnershipErrorCode) {
    super(`ownership transfer refused: ${code}`);
  }
}

// One transaction: the deferred one-Owner constraint lets the new Owner exist before the old one steps down.
export async function transferOwnership(
  actor: Actor,
  memberId: string
): Promise<OwnershipResult> {
  if (!canTransferOwnership(actor.role)) {
    return { kind: "error", code: "forbidden" };
  }

  const { adapter } = await getIam().auth.$context;
  try {
    await adapter.transaction(async (trx) => {
      const promoted = await trx.updateMany({
        model: "member",
        where: [
          { field: "id", value: memberId },
          { field: "organizationId", value: actor.organizationId },
          { field: "role", operator: "in", value: ["admin", "viewer"] },
        ],
        update: { role: "owner" },
      });
      if (promoted !== 1) throw new TransferRefused("invalidInput");

      const demoted = await trx.updateMany({
        model: "member",
        where: [
          { field: "id", value: actor.memberId },
          { field: "organizationId", value: actor.organizationId },
          { field: "role", value: "owner" },
        ],
        update: { role: "admin" },
      });
      if (demoted !== 1) throw new TransferRefused("forbidden");
    });
  } catch (error) {
    if (error instanceof TransferRefused) {
      return { kind: "error", code: error.code };
    }
    throw error;
  }

  return { kind: "ok" };
}
