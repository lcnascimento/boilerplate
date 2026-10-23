"use server";

import { headers } from "next/headers";

import { createOrganizationFor } from "@/shared/auth/organization-creation";
import { organizationSlug } from "@/shared/auth/organization-slug";
import {
  organizationNameInputSchema,
  type OrganizationsResult,
} from "@/shared/auth/organizations";

import {
  requireAuditedActor,
  runOperation,
  serverAction,
} from "./server-action";

const refusals = {
  YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS: "organizationLimit",
} as const;

export const createOrganization = serverAction(
  organizationNameInputSchema,
  async ({ name }): Promise<OrganizationsResult> => {
    const creator = await requireAuditedActor();
    const requestHeaders = new Headers(await headers());

    return runOperation("organizations.create", refusals, () =>
      createOrganizationFor(creator, requestHeaders, {
        name,
        slug: organizationSlug(name),
      })
    );
  }
);
