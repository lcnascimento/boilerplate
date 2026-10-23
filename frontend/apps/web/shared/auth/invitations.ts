import { z } from "zod";

import { memberRoleSchema } from "./members";

export const invitableRoleSchema = memberRoleSchema.exclude(["owner"]);

export type InvitableRole = z.infer<typeof invitableRoleSchema>;

export const invitableRoles = invitableRoleSchema.options;

export const createInvitationInputSchema = z.object({
  email: z.email(),
  role: invitableRoleSchema,
});

export const invitationIdInputSchema = z.object({
  invitationId: z.uuid(),
});

export type InvitationsErrorCode =
  | "invalidInput"
  | "forbidden"
  | "alreadyMember"
  | "alreadyInvited"
  | "invitationLimit"
  | "invitationNotFound"
  | "organizationLimit"
  | "unexpected";

export type InvitationsResult =
  { kind: "ok" } | { kind: "error"; code: InvitationsErrorCode };

export interface SentInvitation {
  id: string;
  email: string;
  role: InvitableRole;
  expiresAt: Date;
}

export interface ReceivedInvitation {
  id: string;
  organizationName: string;
  role: InvitableRole;
  expiresAt: Date;
}

export function pendingInvitations<
  T extends { status: string; createdAt: Date; expiresAt: Date },
>(invitations: readonly T[], now: Date): T[] {
  return invitations
    .filter(({ status, expiresAt }) => status === "pending" && expiresAt > now)
    .sort((a, b) => a.createdAt.getTime() - b.createdAt.getTime());
}
