import { z } from "zod";

const actorIdsSchema = z.object({
  userId: z.uuid().brand<"UserId">(),
  organizationId: z.uuid().brand<"OrganizationId">(),
});

export type ActorIds = z.infer<typeof actorIdsSchema>;

export function parseActorIds(ids: {
  userId: string;
  organizationId: string;
}): ActorIds {
  return actorIdsSchema.parse(ids);
}

export interface PrefixedActorIds {
  userId: `user_${string}`;
  organizationId: `org_${string}`;
}

export function prefixedActorIds({
  userId,
  organizationId,
}: ActorIds): PrefixedActorIds {
  return { userId: `user_${userId}`, organizationId: `org_${organizationId}` };
}
