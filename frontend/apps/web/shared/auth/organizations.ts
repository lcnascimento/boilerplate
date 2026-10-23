import { z } from "zod";

import { ORGANIZATION_NAME_MAX_LENGTH } from "./policy";

export interface Organization {
  id: string;
  name: string;
}

export interface UserOrganizations {
  active: Organization;
  organizations: readonly Organization[];
}

export const createOrganizationInputSchema = z.object({
  name: z.string().trim().min(1).max(ORGANIZATION_NAME_MAX_LENGTH),
});

export type OrganizationsErrorCode =
  "invalidInput" | "organizationLimit" | "unexpected";

export type OrganizationsResult =
  { kind: "ok" } | { kind: "error"; code: OrganizationsErrorCode };
