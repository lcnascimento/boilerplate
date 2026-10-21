import { APIError } from "better-auth/api";
import { z } from "zod";

import { ORGANIZATION_NAME_MAX_LENGTH } from "./policy";

const signUpBodySchema = z.object({
  organizationName: z.string().trim().min(1).max(ORGANIZATION_NAME_MAX_LENGTH),
});

export function signUpOrganizationName(
  endpoint: { body?: unknown } | null
): string | null {
  if (endpoint === null) return null;

  const parsed = signUpBodySchema.safeParse(endpoint.body);
  if (!parsed.success) {
    throw new APIError("BAD_REQUEST", {
      code: "INVALID_ORGANIZATION_NAME",
      message: "Invalid organization name",
    });
  }
  return parsed.data.organizationName;
}
