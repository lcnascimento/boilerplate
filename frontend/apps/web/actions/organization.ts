"use server";

import { redirect } from "next/navigation";

import { transferOwnershipInputSchema } from "@/shared/auth/members";
import {
  transferOwnership as transferOwnershipOf,
  type OwnershipResult,
} from "@/shared/auth/ownership";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { logger } from "@/shared/logging";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

export async function transferOwnership(
  input: unknown
): Promise<OwnershipResult> {
  const parsed = transferOwnershipInputSchema.safeParse(input);
  if (!parsed.success) return { kind: "error", code: "invalidInput" };

  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor")
    return { kind: "error", code: "forbidden" };

  try {
    return await transferOwnershipOf(requestActor.actor, parsed.data.memberId);
  } catch (error) {
    logger.error(
      { err: error, operation: "organization.transferOwnership" },
      "organization operation failed"
    );
    return { kind: "error", code: "unexpected" };
  }
}
