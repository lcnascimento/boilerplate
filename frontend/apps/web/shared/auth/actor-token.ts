import "server-only";

import { prefixedActorIds, type ActorIds } from "./actor-ids";
import { getIam } from "./auth";
import { memberRoleSchema } from "./members";
import { roleNames } from "./roles.gen";

const DEFAULT_TTL_SECONDS = 5 * 60;

export type ActorTokenFailure = "not_a_member" | "unknown_role";

export class ActorTokenError extends Error {
  override readonly name = "ActorTokenError";

  constructor(
    readonly reason: ActorTokenFailure,
    message: string
  ) {
    super(message);
  }
}

export interface MintActorTokenInput extends ActorIds {
  ttlSeconds?: number;
}

export async function mintActorToken({
  ttlSeconds = DEFAULT_TTL_SECONDS,
  ...ids
}: MintActorTokenInput): Promise<string> {
  const { userId, organizationId } = ids;
  const { auth } = getIam();
  const { adapter } = await auth.$context;

  const member = await adapter.findOne<{ role: string }>({
    model: "member",
    where: [
      { field: "organizationId", value: organizationId },
      { field: "userId", value: userId },
    ],
  });
  if (member === null) {
    throw new ActorTokenError(
      "not_a_member",
      `user ${userId} is not a member of organization ${organizationId}`
    );
  }
  const role = memberRoleSchema.safeParse(member.role);
  if (!role.success) {
    throw new ActorTokenError(
      "unknown_role",
      `member role "${member.role}" is not one of ${roleNames.join(", ")}`
    );
  }

  const prefixed = prefixedActorIds(ids);
  const iat = Math.floor(Date.now() / 1000);
  const { token } = await auth.api.signJWT({
    body: {
      payload: {
        sub: prefixed.userId,
        org: prefixed.organizationId,
        role: role.data,
        iat,
        exp: iat + ttlSeconds,
      },
    },
  });
  return token;
}
