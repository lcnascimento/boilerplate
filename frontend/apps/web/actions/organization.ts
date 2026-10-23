"use server";

import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";

import { runAsActor } from "@/shared/auth/audit-connection";
import { getIam } from "@/shared/auth/auth";
import { transferOwnershipInputSchema } from "@/shared/auth/members";
import { deleteOrganizationFor } from "@/shared/auth/organization-deletion";
import {
  canManageOrganization,
  deleteOrganizationInputSchema,
  organizationNameInputSchema,
  type DeletionResult,
  type Organization,
  type RenameResult,
} from "@/shared/auth/organizations";
import {
  transferOwnershipFor,
  type OwnershipResult,
} from "@/shared/auth/ownership";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";

import {
  forbidden,
  logFailure,
  requireActor,
  runOperation,
  serverAction,
  unexpected,
} from "./server-action";

export async function getActiveOrganization(): Promise<Organization> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor") notFound();
  return requestActor.organizations.active;
}

export const transferOwnership = serverAction(
  transferOwnershipInputSchema,
  async ({ memberId }): Promise<OwnershipResult> => {
    const actor = await requireActor();
    if (actor === null) return forbidden;

    try {
      return await runAsActor(actor, new Headers(await headers()), () =>
        transferOwnershipFor(actor, memberId)
      );
    } catch (error) {
      logFailure("organization.transferOwnership", error);
      return unexpected;
    }
  }
);

const renameRefusals = {
  YOU_ARE_NOT_ALLOWED_TO_UPDATE_THIS_ORGANIZATION: "forbidden",
  USER_IS_NOT_A_MEMBER_OF_THE_ORGANIZATION: "forbidden",
} as const;

export const renameOrganization = serverAction(
  organizationNameInputSchema,
  async ({ name }): Promise<RenameResult> => {
    const requestHeaders = await headers();
    const actor = await requireActor();
    if (actor === null || !canManageOrganization(actor.role, "update")) {
      return forbidden;
    }

    return runOperation("organization.rename", renameRefusals, () =>
      getIam().auth.api.updateOrganization({
        headers: requestHeaders,
        body: { organizationId: actor.organizationId, data: { name } },
      })
    );
  }
);

export const deleteOrganization = serverAction(
  deleteOrganizationInputSchema,
  async ({ confirmation }): Promise<DeletionResult> => {
    const actor = await requireActor();
    if (actor === null) return forbidden;

    try {
      return await runAsActor(actor, new Headers(await headers()), () =>
        deleteOrganizationFor(actor, confirmation)
      );
    } catch (error) {
      logFailure("organization.delete", error);
      return unexpected;
    }
  }
);
