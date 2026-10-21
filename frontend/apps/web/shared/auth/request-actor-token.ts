import "server-only";

import { headers } from "next/headers";
import { redirect } from "next/navigation";

import { resolveActiveOrganization } from "./active-organization";
import { annotateActorSpan } from "./actor-span";
import { mintActorToken } from "./actor-token";
import { getIam } from "./auth";
import { SIGN_IN_HREF } from "./routes";

// The GetToken of every @boilerplate/api client a server action builds: one fresh token per call, never cached.
export async function requestActorToken(): Promise<string> {
  const session = await getIam().auth.api.getSession({
    headers: await headers(),
  });
  if (session === null) redirect(SIGN_IN_HREF);

  // With no Organization left, the (dashboard) layout shows the "no Organization" screen.
  const organizations = await resolveActiveOrganization(session.session);
  if (organizations === null) redirect("/");

  const actor = {
    userId: session.user.id,
    organizationId: organizations.active.id,
  };
  annotateActorSpan(actor);
  return mintActorToken(actor);
}
