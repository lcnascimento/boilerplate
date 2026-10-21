import "server-only";

import { redirect } from "next/navigation";

import { resolveActiveOrganization } from "./active-organization";
import { parseActorIds } from "./actor-ids";
import { annotateActorSpan } from "./actor-span";
import { mintActorToken } from "./actor-token";
import { SIGN_IN_HREF } from "./routes";
import { getRequestSession } from "./session";

export async function requestActorToken(): Promise<string> {
  const session = await getRequestSession();
  if (session === null) redirect(SIGN_IN_HREF);

  const organizations = await resolveActiveOrganization(session.session);
  if (organizations === null) redirect("/");

  const actor = parseActorIds({
    userId: session.user.id,
    organizationId: organizations.active.id,
  });
  annotateActorSpan(actor);
  return mintActorToken(actor);
}
