import "server-only";

import { runWithTransaction } from "@better-auth/core/context";

import { runAsActor, type AuditedActor } from "./audit-connection";
import { getIam } from "./auth";

export async function createOrganizationFor(
  creator: AuditedActor,
  headers: Headers,
  body: { name: string; slug: string }
) {
  const { auth } = getIam();
  const { adapter } = await auth.$context;
  return runAsActor(creator, headers, () =>
    runWithTransaction(adapter, () =>
      auth.api.createOrganization({ headers, body })
    )
  );
}
