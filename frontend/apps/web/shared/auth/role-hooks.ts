import { APIError } from "better-auth/api";

import { memberRoleSchema } from "./members";

// Better Auth stores a Member's Roles comma-separated.
function parseRoles(role: string): string[] {
  return role.split(",").map((name) => name.trim());
}

function unknownRole(roles: readonly string[]): APIError | undefined {
  const unknown = roles.filter(
    (name) => !memberRoleSchema.safeParse(name).success
  );
  if (unknown.length === 0) return undefined;
  return new APIError("BAD_REQUEST", {
    code: "UNKNOWN_ROLE",
    message: `Unknown role: ${unknown.join(", ")}`,
  });
}

function ownerAssigned(roles: readonly string[]): APIError | undefined {
  if (!roles.includes("owner")) return undefined;
  return new APIError("BAD_REQUEST", {
    code: "OWNER_NOT_ASSIGNABLE",
    message:
      "Only creating an Organization or transferring its ownership makes an Owner",
  });
}

function ownerChanged(role: string): APIError | undefined {
  if (!parseRoles(role).includes("owner")) return undefined;
  return new APIError("BAD_REQUEST", {
    code: "OWNER_MUST_TRANSFER",
    message: "The Owner must transfer the ownership first",
  });
}

function settle(error: APIError | undefined): Promise<void> {
  return error === undefined ? Promise.resolve() : Promise.reject(error);
}

export function roleHooks(
  hasMembers: (organizationId: string) => Promise<boolean>
) {
  return {
    async beforeAddMember({
      member,
    }: {
      member: { organizationId: string; role: string };
    }): Promise<void> {
      const roles = parseRoles(member.role);
      // Creating an Organization adds its creator as the Owner of an Organization without Members.
      const isCreation = !(await hasMembers(member.organizationId));
      return settle(
        unknownRole(roles) ?? (isCreation ? undefined : ownerAssigned(roles))
      );
    },
    beforeUpdateMemberRole({
      member,
      newRole,
    }: {
      member: { role: string };
      newRole: string;
    }): Promise<void> {
      const roles = parseRoles(newRole);
      return settle(
        unknownRole(roles) ?? ownerAssigned(roles) ?? ownerChanged(member.role)
      );
    },
    beforeRemoveMember({
      member,
    }: {
      member: { role: string };
    }): Promise<void> {
      return settle(ownerChanged(member.role));
    },
    beforeCreateInvitation({
      invitation,
    }: {
      invitation: { role: string };
    }): Promise<void> {
      const roles = parseRoles(invitation.role);
      return settle(unknownRole(roles) ?? ownerAssigned(roles));
    },
  };
}
