import "server-only";

import { prefixedActorIds } from "./actor-ids";
import { getIam } from "./auth";

import type { Actor } from "./actor";

const DEFAULT_TTL_SECONDS = 5 * 60;

export async function mintActorToken(
  actor: Actor,
  ttlSeconds = DEFAULT_TTL_SECONDS
): Promise<string> {
  const prefixed = prefixedActorIds(actor);
  const iat = Math.floor(Date.now() / 1000);
  const { token } = await getIam().auth.api.signJWT({
    body: {
      payload: {
        sub: prefixed.userId,
        org: prefixed.organizationId,
        role: actor.role,
        iat,
        exp: iat + ttlSeconds,
      },
    },
  });
  return token;
}
