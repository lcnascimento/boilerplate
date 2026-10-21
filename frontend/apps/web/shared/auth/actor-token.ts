import "server-only";

import { getIam } from "./auth";
import { roles } from "./roles.gen";

const DEFAULT_TTL_SECONDS = 5 * 60;

type Role = keyof typeof roles;

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

export interface MintActorTokenInput {
  userId: string;
  organizationId: string;
  ttlSeconds?: number;
}

// Better Auth stores a Member's Roles comma-separated, so "owner,admin" is not a Role either.
function isRole(value: string): value is Role {
  return Object.hasOwn(roles, value);
}

export async function mintActorToken({
  userId,
  organizationId,
  ttlSeconds = DEFAULT_TTL_SECONDS,
}: MintActorTokenInput): Promise<string> {
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
  if (!isRole(member.role)) {
    throw new ActorTokenError(
      "unknown_role",
      `member role "${member.role}" is not one of ${Object.keys(roles).join(", ")}`
    );
  }

  const iat = Math.floor(Date.now() / 1000);
  const { token } = await auth.api.signJWT({
    body: {
      payload: {
        sub: `user_${userId}`,
        org: `org_${organizationId}`,
        role: member.role,
        iat,
        exp: iat + ttlSeconds,
      },
    },
  });
  return token;
}
