import "server-only";

import { sql, type Kysely } from "kysely";
import { z } from "zod";

import {
  memberIdSchema,
  organizationIdSchema,
  type OrganizationId,
  type UserId,
} from "./actor-ids";
import { runAsActor, type AuditedActor } from "./audit-connection";
import { getIam } from "./auth";
import { memberRoleSchema } from "./members";
import { deleteOwnedOrganization } from "./organization-deletion";
import { lockOrganizationsOfMember } from "./organization-lock";
import { ACCOUNT_DELETION_ATTEMPTS } from "./policy";

import type { AccountDeletionResult } from "./account";
import type { Organization } from "./organizations";

const membershipSchema = z.object({
  memberId: memberIdSchema,
  organizationId: organizationIdSchema,
  role: memberRoleSchema,
  soleMember: z.boolean(),
});

type Membership = z.infer<typeof membershipSchema>;

const organizationSchema = z.object({
  id: organizationIdSchema,
  name: z.string(),
});

async function membershipsOf(
  connection: Kysely<unknown>,
  userId: UserId
): Promise<Membership[]> {
  const { rows } = await sql`
    SELECT m.id AS "memberId", m."organizationId", m.role,
           (SELECT count(*) FROM member other
             WHERE other."organizationId" = m."organizationId") = 1 AS "soleMember"
      FROM member m
     WHERE m."userId" = ${userId}`.execute(connection);
  return z.array(membershipSchema).parse(rows);
}

async function sharedOwnedOrganizations(
  connection: Kysely<unknown>,
  memberships: readonly Membership[]
): Promise<Organization[]> {
  const ids: OrganizationId[] = memberships
    .filter(({ role, soleMember }) => role === "owner" && !soleMember)
    .map(({ organizationId }) => organizationId);
  if (ids.length === 0) return [];

  const { rows } = await sql`
    SELECT id, name FROM organization
     WHERE id = ANY(${ids}::uuid[])
     ORDER BY name`.execute(connection);
  return z.array(organizationSchema).parse(rows);
}

export async function accountDeletionBlockers(
  userId: UserId
): Promise<Organization[]> {
  const { database } = getIam();
  return sharedOwnedOrganizations(
    database,
    await membershipsOf(database, userId)
  );
}

async function consumeDeletionAttempt(userId: UserId): Promise<boolean> {
  const now = Date.now();
  const windowStart = now - ACCOUNT_DELETION_ATTEMPTS.windowSeconds * 1000;
  const { rows } = await sql`
    INSERT INTO "rateLimit" (key, count, "lastRequest")
    VALUES (${`delete-account:${userId}`}, 1, ${now})
    ON CONFLICT (key) DO UPDATE SET
      count = CASE WHEN "rateLimit"."lastRequest" <= ${windowStart}
                   THEN 1 ELSE "rateLimit".count + 1 END,
      "lastRequest" = CASE WHEN "rateLimit"."lastRequest" <= ${windowStart}
                           THEN ${now} ELSE "rateLimit"."lastRequest" END
    RETURNING count::int AS attempts`.execute(getIam().database);
  const [{ attempts }] = z
    .tuple([z.object({ attempts: z.number() })])
    .parse(rows);
  return attempts <= ACCOUNT_DELETION_ATTEMPTS.max;
}

async function passwordMatches(
  userId: UserId,
  password: string
): Promise<boolean> {
  const context = await getIam().auth.$context;
  const account = await context.internalAdapter.findCredentialAccount(userId);
  if (typeof account?.password !== "string") return false;
  return context.password.verify({ hash: account.password, password });
}

export async function deleteAccount(
  actor: AuditedActor,
  headers: Headers,
  password: string
): Promise<AccountDeletionResult> {
  const { userId } = actor;
  if (!(await consumeDeletionAttempt(userId))) {
    return { kind: "error", code: "tooManyAttempts" };
  }
  if (!(await passwordMatches(userId, password))) {
    return { kind: "error", code: "invalidPassword" };
  }

  return runAsActor(actor, headers, () =>
    getIam()
      .database.transaction()
      .execute(async (trx): Promise<AccountDeletionResult> => {
        await lockOrganizationsOfMember(trx, userId);
        const memberships = await membershipsOf(trx, userId);
        const blockers = await sharedOwnedOrganizations(trx, memberships);
        if (blockers.length > 0) {
          return {
            kind: "error",
            code: "ownsSharedOrganizations",
            organizations: blockers,
          };
        }

        for (const membership of memberships.filter(
          ({ soleMember }) => soleMember
        )) {
          const deletion = await deleteOwnedOrganization(trx, membership);
          if (deletion !== "deleted") {
            throw new Error(
              `sole-member organization ${membership.organizationId} was not deleted: ${deletion}`
            );
          }
        }

        await sql`DELETE FROM "user" WHERE id = ${userId}`.execute(trx);
        return { kind: "ok" };
      })
  );
}
