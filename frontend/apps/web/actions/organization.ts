"use server";

import { APIError } from "better-auth/api";
import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";

import { runAsActor } from "@/shared/auth/audit-connection";
import { getIam } from "@/shared/auth/auth";
import { transferOwnershipInputSchema } from "@/shared/auth/members";
import { deleteOrganization as deleteOrganizationOf } from "@/shared/auth/organization-deletion";
import {
  canManageOrganization,
  deleteOrganizationInputSchema,
  organizationNameInputSchema,
  type DeletionResult,
  type Organization,
  type RenameResult,
} from "@/shared/auth/organizations";
import {
  transferOwnership as transferOwnershipOf,
  type OwnershipResult,
} from "@/shared/auth/ownership";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { logger } from "@/shared/logging";

import type { Actor } from "@/shared/auth/actor";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

const forbidden = { kind: "error", code: "forbidden" } as const;
const invalidInput = { kind: "error", code: "invalidInput" } as const;
const unexpected = { kind: "error", code: "unexpected" } as const;

async function requireActor(): Promise<Actor | null> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  return requestActor.kind === "actor" ? requestActor.actor : null;
}

function logFailure(operation: string, error: unknown) {
  logger.error({ err: error, operation }, "organization operation failed");
}

export async function getActiveOrganization(): Promise<Organization> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor") notFound();
  return requestActor.organizations.active;
}

export async function transferOwnership(
  input: unknown
): Promise<OwnershipResult> {
  const parsed = transferOwnershipInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const actor = await requireActor();
  if (actor === null) return forbidden;

  try {
    return await runAsActor(actor, new Headers(await headers()), () =>
      transferOwnershipOf(actor, parsed.data.memberId)
    );
  } catch (error) {
    logFailure("organization.transferOwnership", error);
    return unexpected;
  }
}

const refusedRenameCodes = new Set([
  "YOU_ARE_NOT_ALLOWED_TO_UPDATE_THIS_ORGANIZATION",
  "USER_IS_NOT_A_MEMBER_OF_THE_ORGANIZATION",
]);

export async function renameOrganization(
  input: unknown
): Promise<RenameResult> {
  const parsed = organizationNameInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null || !canManageOrganization(actor.role, "update")) {
    return forbidden;
  }

  try {
    await getIam().auth.api.updateOrganization({
      headers: requestHeaders,
      body: {
        organizationId: actor.organizationId,
        data: { name: parsed.data.name },
      },
    });
    return { kind: "ok" };
  } catch (error) {
    if (
      error instanceof APIError &&
      typeof error.body?.code === "string" &&
      refusedRenameCodes.has(error.body.code)
    ) {
      return forbidden;
    }
    logFailure("organization.rename", error);
    return unexpected;
  }
}

export async function deleteOrganization(
  input: unknown
): Promise<DeletionResult> {
  const parsed = deleteOrganizationInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const actor = await requireActor();
  if (actor === null) return forbidden;

  try {
    return await runAsActor(actor, new Headers(await headers()), () =>
      deleteOrganizationOf(actor, parsed.data.confirmation)
    );
  } catch (error) {
    logFailure("organization.delete", error);
    return unexpected;
  }
}
