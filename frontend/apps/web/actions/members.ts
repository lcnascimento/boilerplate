"use server";

import { APIError } from "better-auth/api";
import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";

import { getIam } from "@/shared/auth/auth";
import {
  addMemberInputSchema,
  can,
  canAssign,
  memberRoleSchema,
  removeMemberInputSchema,
  updateMemberRoleInputSchema,
  type Member,
  type MembersErrorCode,
  type MembersResult,
} from "@/shared/auth/members";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { logger } from "@/shared/logging";

import type { Actor } from "@/shared/auth/actor";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

async function requireActor(): Promise<Actor | null> {
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  return requestActor.kind === "actor" ? requestActor.actor : null;
}

const errorCodes: Record<string, MembersErrorCode> = {
  USER_NOT_FOUND: "userNotFound",
  USER_IS_ALREADY_A_MEMBER_OF_THIS_ORGANIZATION: "alreadyMember",
  YOU_CANNOT_LEAVE_THE_ORGANIZATION_AS_THE_ONLY_OWNER: "lastOwner",
  YOU_CANNOT_LEAVE_THE_ORGANIZATION_WITHOUT_AN_OWNER: "lastOwner",
  YOU_ARE_NOT_ALLOWED_TO_UPDATE_THIS_MEMBER: "forbidden",
  YOU_ARE_NOT_ALLOWED_TO_DELETE_THIS_MEMBER: "forbidden",
  YOU_ARE_NOT_A_MEMBER_OF_THIS_ORGANIZATION: "forbidden",
  MEMBER_NOT_FOUND: "invalidInput",
  ROLE_NOT_FOUND: "invalidInput",
  UNKNOWN_ROLE: "invalidInput",
};

async function runMembersOperation(
  operation: string,
  run: () => Promise<unknown>
): Promise<MembersResult> {
  try {
    await run();
    return { kind: "ok" };
  } catch (error) {
    const code =
      error instanceof APIError && typeof error.body?.code === "string"
        ? errorCodes[error.body.code]
        : undefined;
    if (code !== undefined) return { kind: "error", code };

    logger.error({ err: error, operation }, "members operation failed");
    return { kind: "error", code: "unexpected" };
  }
}

const forbidden = { kind: "error", code: "forbidden" } as const;
const invalidInput = { kind: "error", code: "invalidInput" } as const;

export async function listMembers(): Promise<{
  actor: Member;
  members: Member[];
}> {
  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null) notFound();

  const { members } = await getIam().auth.api.listMembers({
    headers: requestHeaders,
    query: {
      organizationId: actor.organizationId,
      sortBy: "createdAt",
      sortDirection: "asc",
    },
  });
  const parsed = members.map(({ id, role, user }): Member => ({
    id,
    name: user.name,
    email: user.email,
    role: memberRoleSchema.parse(role),
  }));
  const self = parsed.find(({ id }) => id === actor.memberId);
  if (self === undefined) notFound();

  return { actor: self, members: parsed };
}

export async function addMember(input: unknown): Promise<MembersResult> {
  const parsed = addMemberInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const actor = await requireActor();
  // auth.api.addMember checks no Permission at all: this action is the only gate.
  if (
    actor === null ||
    !can(actor.role, "create") ||
    !canAssign(actor.role, parsed.data.role)
  ) {
    return forbidden;
  }

  const { auth } = getIam();
  const context = await auth.$context;
  const user = await context.internalAdapter.findUserByEmail(parsed.data.email);
  if (user === null) return { kind: "error", code: "userNotFound" };

  return runMembersOperation("members.add", () =>
    auth.api.addMember({
      body: {
        userId: user.user.id,
        role: parsed.data.role,
        organizationId: actor.organizationId,
      },
    })
  );
}

export async function updateMemberRole(input: unknown): Promise<MembersResult> {
  const parsed = updateMemberRoleInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null) return forbidden;

  return runMembersOperation("members.updateRole", () =>
    getIam().auth.api.updateMemberRole({
      headers: requestHeaders,
      body: { ...parsed.data, organizationId: actor.organizationId },
    })
  );
}

export async function removeMember(input: unknown): Promise<MembersResult> {
  const parsed = removeMemberInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null) return forbidden;

  return runMembersOperation("members.remove", () =>
    getIam().auth.api.removeMember({
      headers: requestHeaders,
      body: {
        memberIdOrEmail: parsed.data.memberId,
        organizationId: actor.organizationId,
      },
    })
  );
}

export async function leaveOrganization(): Promise<MembersResult> {
  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null) return forbidden;

  return runMembersOperation("members.leave", () =>
    getIam().auth.api.leaveOrganization({
      headers: requestHeaders,
      body: { organizationId: actor.organizationId },
    })
  );
}
