"use server";

import { headers } from "next/headers";
import { notFound } from "next/navigation";

import { getIam } from "@/shared/auth/auth";
import {
  memberRoleSchema,
  removeMemberInputSchema,
  updateMemberRoleInputSchema,
  type Member,
  type MembersErrorCode,
  type MembersResult,
} from "@/shared/auth/members";

import {
  forbidden,
  requireActor,
  runOperation,
  serverAction,
} from "./server-action";

const refusals: Record<string, MembersErrorCode> = {
  OWNER_NOT_ASSIGNABLE: "ownerNotAssignable",
  OWNER_MUST_TRANSFER: "ownerMustTransfer",
  YOU_CANNOT_LEAVE_THE_ORGANIZATION_AS_THE_ONLY_OWNER: "ownerMustTransfer",
  YOU_CANNOT_LEAVE_THE_ORGANIZATION_WITHOUT_AN_OWNER: "ownerMustTransfer",
  YOU_ARE_NOT_ALLOWED_TO_UPDATE_THIS_MEMBER: "forbidden",
  YOU_ARE_NOT_ALLOWED_TO_DELETE_THIS_MEMBER: "forbidden",
  YOU_ARE_NOT_A_MEMBER_OF_THIS_ORGANIZATION: "forbidden",
  MEMBER_NOT_FOUND: "invalidInput",
  ROLE_NOT_FOUND: "invalidInput",
  UNKNOWN_ROLE: "invalidInput",
};

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

export const updateMemberRole = serverAction(
  updateMemberRoleInputSchema,
  async (update): Promise<MembersResult> => {
    const requestHeaders = await headers();
    const actor = await requireActor();
    if (actor === null) return forbidden;

    return runOperation("members.updateRole", refusals, () =>
      getIam().auth.api.updateMemberRole({
        headers: requestHeaders,
        body: { ...update, organizationId: actor.organizationId },
      })
    );
  }
);

export const removeMember = serverAction(
  removeMemberInputSchema,
  async ({ memberId }): Promise<MembersResult> => {
    const requestHeaders = await headers();
    const actor = await requireActor();
    if (actor === null) return forbidden;

    return runOperation("members.remove", refusals, () =>
      getIam().auth.api.removeMember({
        headers: requestHeaders,
        body: {
          memberIdOrEmail: memberId,
          organizationId: actor.organizationId,
        },
      })
    );
  }
);

export async function leaveOrganization(): Promise<MembersResult> {
  const requestHeaders = await headers();
  const actor = await requireActor();
  if (actor === null) return forbidden;

  return runOperation("members.leave", refusals, () =>
    getIam().auth.api.leaveOrganization({
      headers: requestHeaders,
      body: { organizationId: actor.organizationId },
    })
  );
}
