import { z } from "zod";

import { roles } from "./roles.gen";

export const memberRoles = [
  "owner",
  "admin",
  "viewer",
] as const satisfies (keyof typeof roles)[];

export const memberRoleSchema = z.enum(memberRoles);

export type MemberRole = z.infer<typeof memberRoleSchema>;

export const addMemberInputSchema = z.object({
  email: z.email(),
  role: memberRoleSchema,
});

export type AddMemberInput = z.infer<typeof addMemberInputSchema>;

export const updateMemberRoleInputSchema = z.object({
  memberId: z.string().min(1),
  role: memberRoleSchema,
});

export const removeMemberInputSchema = z.object({
  memberId: z.string().min(1),
});

export type MembersErrorCode =
  | "invalidInput"
  | "forbidden"
  | "userNotFound"
  | "alreadyMember"
  | "lastOwner"
  | "unexpected";

export type MembersResult =
  { kind: "ok" } | { kind: "error"; code: MembersErrorCode };

export interface Member {
  id: string;
  name: string;
  email: string;
  role: MemberRole;
}

export interface MemberRow {
  member: Member;
  canChangeRole: boolean;
  canRemove: boolean;
}

export interface MemberPolicy {
  canAdd: boolean;
  assignableRoles: MemberRole[];
  leave: "allowed" | "lastOwner";
  rows: MemberRow[];
}

type MemberAction = "create" | "update" | "delete";

export function can(role: MemberRole, action: MemberAction): boolean {
  return roles[role].authorize({ member: [action] }).success;
}

export function canAssign(actorRole: MemberRole, role: MemberRole): boolean {
  return role !== "owner" || actorRole === "owner";
}

export function memberPolicy(
  actor: Pick<Member, "id" | "role">,
  members: readonly Member[]
): MemberPolicy {
  const isLastOwner = (member: Member) =>
    member.role === "owner" &&
    members.filter(({ role }) => role === "owner").length === 1;
  const reaches = (member: Member) =>
    canAssign(actor.role, member.role) && !isLastOwner(member);

  return {
    canAdd: can(actor.role, "create"),
    assignableRoles: can(actor.role, "update")
      ? memberRoles.filter((role) => canAssign(actor.role, role))
      : [],
    leave: members.some(
      (member) => member.id === actor.id && isLastOwner(member)
    )
      ? "lastOwner"
      : "allowed",
    rows: members.map((member) => ({
      member,
      canChangeRole: can(actor.role, "update") && reaches(member),
      canRemove:
        can(actor.role, "delete") && member.id !== actor.id && reaches(member),
    })),
  };
}
