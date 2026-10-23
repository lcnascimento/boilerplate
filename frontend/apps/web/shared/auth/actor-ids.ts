import { z } from "zod";

export const userIdSchema = z.uuid().brand<"UserId">();

export const organizationIdSchema = z.uuid().brand<"OrganizationId">();

const actorIdsSchema = z.object({
  userId: userIdSchema,
  organizationId: organizationIdSchema,
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
