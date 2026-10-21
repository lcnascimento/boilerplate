import "server-only";

import { headers } from "next/headers";
import { redirect } from "next/navigation";

import { mintActorToken } from "./actor-token";
import { getIam } from "./auth";
import { SIGN_IN_HREF } from "./routes";

// The GetToken of every @boilerplate/api client a server action builds: one fresh token per call, never cached.
export async function requestActorToken(): Promise<string> {
  const session = await getIam().auth.api.getSession({
    headers: await headers(),
  });
  if (session === null) redirect(SIGN_IN_HREF);

  const { activeOrganizationId } = session.session;
  if (!activeOrganizationId) {
    throw new Error(`user ${session.user.id} has no active organization`);
  }

  return mintActorToken({
    userId: session.user.id,
    organizationId: activeOrganizationId,
  });
}
