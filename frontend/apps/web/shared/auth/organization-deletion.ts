import "server-only";

import { getIam, type IamAdapter } from "./auth";
import { canManageOrganization, type DeletionResult } from "./organizations";

import type { Actor } from "./actor";
import type { Where } from "better-auth";

interface OrganizationOwner {
  memberId: string;
  organizationId: string;
}

type OwnedOrganizationDeletion = "deleted" | "notOwner" | "notMatched";

export async function deleteOwnedOrganization(
  trx: IamAdapter,
  { memberId, organizationId }: OrganizationOwner,
  confirmation?: { name: string }
): Promise<OwnedOrganizationDeletion> {
  const owner = await trx.findOne({
    model: "member",
    where: [
      { field: "id", value: memberId },
      { field: "organizationId", value: organizationId },
      { field: "role", value: "owner" },
    ],
  });
  if (owner === null) return "notOwner";

  const where: Where[] = [{ field: "id", value: organizationId }];
  if (confirmation !== undefined) {
    where.push({ field: "name", value: confirmation.name });
  }
  const deleted = await trx.deleteMany({ model: "organization", where });
  return deleted === 1 ? "deleted" : "notMatched";
}

export async function deleteOrganization(
  actor: Actor,
  confirmation: string
): Promise<DeletionResult> {
  if (!canManageOrganization(actor.role, "delete")) {
    return { kind: "error", code: "forbidden" };
  }

  const { adapter } = await getIam().auth.$context;
  const deletion = await adapter.transaction((trx) =>
    deleteOwnedOrganization(trx, actor, { name: confirmation })
  );
  switch (deletion) {
    case "deleted":
      return { kind: "ok" };
    case "notOwner":
      return { kind: "error", code: "forbidden" };
    case "notMatched":
      return { kind: "error", code: "nameMismatch" };
  }
}
