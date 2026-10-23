"use server";

import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";

import { getIam } from "@/shared/auth/auth";
import { acceptInvitationFor } from "@/shared/auth/invitation-acceptance";
import {
  createInvitationInputSchema,
  invitableRoleSchema,
  invitationIdInputSchema,
  pendingInvitations,
  type InvitationsErrorCode,
  type InvitationsResult,
  type ReceivedInvitation,
  type SentInvitation,
} from "@/shared/auth/invitations";
import { getRequestActor } from "@/shared/auth/request-actor";
import { SIGN_IN_HREF } from "@/shared/auth/routes";

import {
  forbidden,
  requireAuditedActor,
  requireSession,
  runOperation,
  serverAction,
} from "./server-action";

const refusals: Record<string, InvitationsErrorCode> = {
  USER_IS_ALREADY_A_MEMBER_OF_THIS_ORGANIZATION: "alreadyMember",
  USER_IS_ALREADY_INVITED_TO_THIS_ORGANIZATION: "alreadyInvited",
  INVITATION_LIMIT_REACHED: "invitationLimit",
  INVITATION_NOT_FOUND: "invitationNotFound",
  YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS: "organizationLimit",
  YOU_ARE_NOT_ALLOWED_TO_INVITE_USERS_TO_THIS_ORGANIZATION: "forbidden",
  YOU_ARE_NOT_ALLOWED_TO_INVITE_USER_WITH_THIS_ROLE: "forbidden",
  YOU_ARE_NOT_ALLOWED_TO_CANCEL_THIS_INVITATION: "forbidden",
  YOU_ARE_NOT_THE_RECIPIENT_OF_THE_INVITATION: "forbidden",
  MEMBER_NOT_FOUND: "forbidden",
  OWNER_NOT_ASSIGNABLE: "invalidInput",
  UNKNOWN_ROLE: "invalidInput",
  INVALID_EMAIL: "invalidInput",
};

export async function listSentInvitations(): Promise<SentInvitation[]> {
  const requestHeaders = await headers();
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor") notFound();

  const invitations = await getIam().auth.api.listInvitations({
    headers: requestHeaders,
    query: { organizationId: requestActor.actor.organizationId },
  });
  return pendingInvitations(invitations, new Date()).map(
    ({ id, email, role, expiresAt }) => ({
      id,
      email,
      role: invitableRoleSchema.parse(role),
      expiresAt,
    })
  );
}

export async function listReceivedInvitations(): Promise<ReceivedInvitation[]> {
  const session = await requireSession();

  // Better Auth requires a verified email when it reads the session, and nothing here verifies emails.
  const invitations = await getIam().auth.api.listUserInvitations({
    query: { email: session.user.email },
  });
  return pendingInvitations(invitations, new Date()).map(
    ({ id, organizationName, role, expiresAt }) => ({
      id,
      organizationName,
      role: invitableRoleSchema.parse(role),
      expiresAt,
    })
  );
}

export const createInvitation = serverAction(
  createInvitationInputSchema,
  async (invitation): Promise<InvitationsResult> => {
    const requestHeaders = await headers();
    const requestActor = await getRequestActor();
    if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
    if (requestActor.kind !== "actor") return forbidden;

    return runOperation("invitations.create", refusals, () =>
      getIam().auth.api.createInvitation({
        headers: requestHeaders,
        body: {
          ...invitation,
          organizationId: requestActor.actor.organizationId,
        },
      })
    );
  }
);

export const cancelInvitation = serverAction(
  invitationIdInputSchema,
  async (body): Promise<InvitationsResult> => {
    const requestHeaders = await headers();
    await requireSession();

    return runOperation("invitations.cancel", refusals, () =>
      getIam().auth.api.cancelInvitation({ headers: requestHeaders, body })
    );
  }
);

export const acceptInvitation = serverAction(
  invitationIdInputSchema,
  async ({ invitationId }): Promise<InvitationsResult> => {
    const requestHeaders = new Headers(await headers());
    const acceptor = await requireAuditedActor();

    return runOperation("invitations.accept", refusals, () =>
      acceptInvitationFor(acceptor, requestHeaders, invitationId)
    );
  }
);

export const rejectInvitation = serverAction(
  invitationIdInputSchema,
  async (body): Promise<InvitationsResult> => {
    const requestHeaders = await headers();
    await requireSession();

    return runOperation("invitations.reject", refusals, () =>
      getIam().auth.api.rejectInvitation({ headers: requestHeaders, body })
    );
  }
);
