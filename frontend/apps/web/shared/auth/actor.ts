import { sql, type Kysely } from "kysely";
import { z } from "zod";

import { organizationIdSchema, userIdSchema, type ActorIds } from "./actor-ids";
import { memberRoleSchema } from "./members";

const userMemberSchema = z
  .object({
    memberId: z.uuid().brand<"MemberId">(),
    role: memberRoleSchema,
    organizationId: organizationIdSchema,
    organizationName: z.string(),
  })
  .transform(({ organizationId, organizationName, ...member }) => ({
    ...member,
    organization: { id: organizationId, name: organizationName },
  }));

export type UserMember = z.infer<typeof userMemberSchema>;

export interface Actor
  extends ActorIds, Pick<UserMember, "memberId" | "role"> {}

export async function listMembersOldestFirst(
  database: Kysely<unknown>,
  userId: string
): Promise<UserMember[]> {
  const { rows } = await sql`
    SELECT m.id AS "memberId", m.role, o.id AS "organizationId", o.name AS "organizationName"
      FROM member m
      JOIN organization o ON o.id = m."organizationId"
     WHERE m."userId" = ${userId}
     ORDER BY m."createdAt", o.id`.execute(database);
  return z.array(userMemberSchema).parse(rows);
}

export function actorOf(userId: string, member: UserMember): Actor {
  return {
    userId: userIdSchema.parse(userId),
    organizationId: member.organization.id,
    memberId: member.memberId,
    role: member.role,
  };
}

export async function findActor(
  database: Kysely<unknown>,
  { userId, organizationId }: { userId: string; organizationId: string }
): Promise<Actor> {
  const members = await listMembersOldestFirst(database, userId);
  const member = members.find(
    ({ organization }) => organization.id === organizationId
  );
  if (member === undefined) {
    throw new Error(
      `user ${userId} is not a member of organization ${organizationId}`
    );
  }
  return actorOf(userId, member);
}
