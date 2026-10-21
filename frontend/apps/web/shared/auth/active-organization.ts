import "server-only";

import { getIam } from "./auth";
import {
  pickActiveOrganization,
  type Organization,
  type UserOrganizations,
} from "./organizations";

interface ActiveSession {
  token: string;
  userId: string;
  activeOrganizationId?: string | null;
}

export async function resolveActiveOrganization({
  token,
  userId,
  activeOrganizationId = null,
}: ActiveSession): Promise<UserOrganizations | null> {
  const { auth, database } = getIam();
  const { rows: organizationsOldestMembershipFirst } =
    await database.query<Organization>(
      `SELECT o.id, o.name
       FROM member m
       JOIN organization o ON o.id = m."organizationId"
      WHERE m."userId" = $1
      ORDER BY m."createdAt", o.id`,
      [userId]
    );

  const active = pickActiveOrganization(
    organizationsOldestMembershipFirst,
    activeOrganizationId
  );
  const resolvedId = active?.id ?? null;
  if (resolvedId !== activeOrganizationId) {
    const { internalAdapter } = await auth.$context;
    await internalAdapter.updateSession(token, {
      activeOrganizationId: resolvedId,
    });
  }

  return active === null
    ? null
    : { active, organizations: organizationsOldestMembershipFirst };
}
