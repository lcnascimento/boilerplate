"use server";

import { APIError } from "better-auth/api";
import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";

import { organizationIdSchema, userIdSchema } from "@/shared/auth/actor-ids";
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
import { getRequestSession } from "@/shared/auth/session";
import { logger } from "@/shared/logging";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

const errorCodes: Record<string, InvitationsErrorCode> = {
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

async function runInvitationsOperation(
  operation: string,
  run: () => Promise<unknown>
): Promise<InvitationsResult> {
  try {
    await run();
    return { kind: "ok" };
  } catch (error) {
    const code =
      error instanceof APIError && typeof error.body?.code === "string"
        ? errorCodes[error.body.code]
        : undefined;
    if (code !== undefined) return { kind: "error", code };

    logger.error({ err: error, operation }, "invitations operation failed");
    return { kind: "error", code: "unexpected" };
  }
}

const invalidInput = { kind: "error", code: "invalidInput" } as const;

async function requireSignedIn() {
  const session = await getRequestSession();
  if (session === null) redirect(SIGN_IN_HREF);
  return session;
}

export async function listSentInvitations(): Promise<SentInvitation[]> {
  const requestHeaders = await headers();
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor") notFound();

  const invitations = await getIam().auth.api.listInvitations({
    headers: requestHeaders,
    query: { organizationId: requestActor.actor.organizationId },
  });
  return pendingInvitations(invitations, new Date())
    .sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime())
    .map(({ id, email, role, expiresAt }) => ({
      id,
      email,
      role: invitableRoleSchema.parse(role),
      expiresAt,
    }));
}

export async function listReceivedInvitations(): Promise<ReceivedInvitation[]> {
  const session = await requireSignedIn();

  // Better Auth requires a verified email when it reads the session, and nothing here verifies emails.
  const invitations = await getIam().auth.api.listUserInvitations({
    query: { email: session.user.email },
  });
  return pendingInvitations(invitations, new Date())
    .sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime())
    .map(({ id, organizationName, role, expiresAt }) => ({
      id,
      organizationName,
      role: invitableRoleSchema.parse(role),
      expiresAt,
    }));
}

export async function createInvitation(
  input: unknown
): Promise<InvitationsResult> {
  const parsed = createInvitationInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  const requestActor = await getRequestActor();
  if (requestActor.kind === "signed-out") redirect(SIGN_IN_HREF);
  if (requestActor.kind !== "actor")
    return { kind: "error", code: "forbidden" };

  return runInvitationsOperation("invitations.create", () =>
    getIam().auth.api.createInvitation({
      headers: requestHeaders,
      body: {
        ...parsed.data,
        organizationId: requestActor.actor.organizationId,
      },
    })
  );
}

export async function cancelInvitation(
  input: unknown
): Promise<InvitationsResult> {
  const parsed = invitationIdInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  await requireSignedIn();

  return runInvitationsOperation("invitations.cancel", () =>
    getIam().auth.api.cancelInvitation({
      headers: requestHeaders,
      body: parsed.data,
    })
  );
}

export async function acceptInvitation(
  input: unknown
): Promise<InvitationsResult> {
  const parsed = invitationIdInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = new Headers(await headers());
  const { user, session } = await requireSignedIn();

  return runInvitationsOperation("invitations.accept", () =>
    acceptInvitationFor(
      {
        userId: userIdSchema.parse(user.id),
        organizationId:
          session.activeOrganizationId == null
            ? null
            : organizationIdSchema.parse(session.activeOrganizationId),
      },
      requestHeaders,
      parsed.data.invitationId
    )
  );
}

export async function rejectInvitation(
  input: unknown
): Promise<InvitationsResult> {
  const parsed = invitationIdInputSchema.safeParse(input);
  if (!parsed.success) return invalidInput;

  const requestHeaders = await headers();
  await requireSignedIn();

  return runInvitationsOperation("invitations.reject", () =>
    getIam().auth.api.rejectInvitation({
      headers: requestHeaders,
      body: parsed.data,
    })
  );
}
