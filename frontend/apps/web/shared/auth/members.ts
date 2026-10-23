import { z } from "zod";

import { canManageOrganization } from "./organizations";
import { roleNames, roles, type Role } from "./roles.gen";

export const memberRoleSchema = z.enum(roleNames);

export type MemberRole = Role;

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

export const transferOwnershipInputSchema = z.object({
  memberId: z.uuid(),
});

export type MembersErrorCode =
  | "invalidInput"
  | "forbidden"
  | "userNotFound"
  | "alreadyMember"
  | "ownerNotAssignable"
  | "ownerMustTransfer"
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
  rows: MemberRow[];
}

export type OwnershipPolicy =
  | { kind: "canTransfer"; candidates: Member[] }
  | { kind: "ownedBy"; owner: Member };

type MemberAction = "create" | "update" | "delete";

export function can(role: MemberRole, action: MemberAction): boolean {
  return roles[role].authorize({ member: [action] }).success;
}

export const assignableRoles = roleNames.filter((role) => role !== "owner");

export function memberPolicy(
  actor: Pick<Member, "id" | "role">,
  members: readonly Member[]
): MemberPolicy {
  const canActOn = (member: Member) => member.role !== "owner";

  return {
    canAdd: can(actor.role, "create"),
    assignableRoles: can(actor.role, "update") ? assignableRoles : [],
    rows: members.map((member) => ({
      member,
      canChangeRole: can(actor.role, "update") && canActOn(member),
      canRemove:
        can(actor.role, "delete") && member.id !== actor.id && canActOn(member),
    })),
  };
}

export function ownershipPolicy(
  actor: Pick<Member, "id" | "role">,
  members: readonly Member[]
): OwnershipPolicy {
  if (canManageOrganization(actor.role, "transfer")) {
    return {
      kind: "canTransfer",
      candidates: members.filter(({ id }) => id !== actor.id),
    };
  }

  const owner = members.find(({ role }) => role === "owner");
  if (owner === undefined) throw new Error("the Organization has no Owner");
  return { kind: "ownedBy", owner };
}
