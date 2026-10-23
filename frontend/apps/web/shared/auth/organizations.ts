import { z } from "zod";

import { ORGANIZATION_NAME_MAX_LENGTH } from "./policy";
import { roles, type Role } from "./roles.gen";

export interface Organization {
  id: string;
  name: string;
}

export interface UserOrganizations {
  active: Organization;
  organizations: readonly Organization[];
}

export const organizationNameInputSchema = z.object({
  name: z.string().trim().min(1).max(ORGANIZATION_NAME_MAX_LENGTH),
});

export const deleteOrganizationInputSchema = z.object({
  confirmation: z.string(),
});

export type OrganizationsErrorCode =
  "invalidInput" | "organizationLimit" | "unexpected";

export type OrganizationsResult =
  { kind: "ok" } | { kind: "error"; code: OrganizationsErrorCode };

export type RenameErrorCode = "invalidInput" | "forbidden" | "unexpected";

export type RenameResult =
  { kind: "ok" } | { kind: "error"; code: RenameErrorCode };

export type DeletionErrorCode = RenameErrorCode | "nameMismatch";

export type DeletionResult =
  { kind: "ok" } | { kind: "error"; code: DeletionErrorCode };

type OrganizationAction = "update" | "delete" | "transfer";

export function canManageOrganization(
  role: Role,
  action: OrganizationAction
): boolean {
  return roles[role].authorize({ organization: [action] }).success;
}

export interface OrganizationSettingsPolicy {
  canRename: boolean;
  canDelete: boolean;
  leave: "allowed" | "ownerMustTransfer";
}

export function organizationSettingsPolicy(
  role: Role
): OrganizationSettingsPolicy {
  return {
    canRename: canManageOrganization(role, "update"),
    canDelete: canManageOrganization(role, "delete"),
    leave: role === "owner" ? "ownerMustTransfer" : "allowed",
  };
}
