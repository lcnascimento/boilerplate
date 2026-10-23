"use server";

import { APIError } from "better-auth/api";
import { headers } from "next/headers";
import { redirect } from "next/navigation";

import { getIam } from "@/shared/auth/auth";
import { organizationSlug } from "@/shared/auth/organization-slug";
import {
  createOrganizationInputSchema,
  type OrganizationsErrorCode,
  type OrganizationsResult,
} from "@/shared/auth/organizations";
import { SIGN_IN_HREF } from "@/shared/auth/routes";
import { getRequestSession } from "@/shared/auth/session";
import { logger } from "@/shared/logging";

// Next exposes every Server Action as a public endpoint, so arguments arrive as `unknown`.

const errorCodes: Record<string, OrganizationsErrorCode> = {
  YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS: "organizationLimit",
};

export async function createOrganization(
  input: unknown
): Promise<OrganizationsResult> {
  const parsed = createOrganizationInputSchema.safeParse(input);
  if (!parsed.success) return { kind: "error", code: "invalidInput" };

  const requestHeaders = await headers();
  if ((await getRequestSession()) === null) redirect(SIGN_IN_HREF);

  try {
    await getIam().auth.api.createOrganization({
      headers: requestHeaders,
      body: {
        name: parsed.data.name,
        slug: organizationSlug(parsed.data.name),
      },
    });
    return { kind: "ok" };
  } catch (error) {
    const code =
      error instanceof APIError && typeof error.body?.code === "string"
        ? errorCodes[error.body.code]
        : undefined;
    if (code !== undefined) return { kind: "error", code };

    logger.error({ err: error }, "creating an organization failed");
    return { kind: "error", code: "unexpected" };
  }
}
