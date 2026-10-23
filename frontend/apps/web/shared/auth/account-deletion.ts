import "server-only";

import { getIam, type IamAdapter } from "./auth";
import { deleteOwnedOrganization } from "./organization-deletion";

import type { AccountDeletionResult } from "./account";
import type { UserId } from "./actor-ids";
import type { Organization } from "./organizations";

interface Membership {
  memberId: string;
  organizationId: string;
  role: string;
  soleMember: boolean;
}

async function membershipsOf(
  adapter: IamAdapter,
  userId: UserId
): Promise<Membership[]> {
  const members = await adapter.findMany<{
    id: string;
    organizationId: string;
    role: string;
  }>({ model: "member", where: [{ field: "userId", value: userId }] });

  return Promise.all(
    members.map(async ({ id, organizationId, role }) => ({
      memberId: id,
      organizationId,
      role,
      soleMember:
        (await adapter.count({
          model: "member",
          where: [{ field: "organizationId", value: organizationId }],
        })) === 1,
    }))
  );
}

async function sharedOwnedOrganizations(
  adapter: IamAdapter,
  memberships: readonly Membership[]
): Promise<Organization[]> {
  const ids = memberships
    .filter(({ role, soleMember }) => role === "owner" && !soleMember)
    .map(({ organizationId }) => organizationId);
  if (ids.length === 0) return [];

  const organizations = await adapter.findMany<Organization>({
    model: "organization",
    where: [{ field: "id", operator: "in", value: ids }],
    sortBy: { field: "name", direction: "asc" },
  });
  return organizations.map(({ id, name }) => ({ id, name }));
}

export async function accountDeletionBlockers(
  userId: UserId
): Promise<Organization[]> {
  const { adapter } = await getIam().auth.$context;
  return sharedOwnedOrganizations(
    adapter,
    await membershipsOf(adapter, userId)
  );
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
  userId: UserId,
  password: string
): Promise<AccountDeletionResult> {
  if (!(await passwordMatches(userId, password))) {
    return { kind: "error", code: "invalidPassword" };
  }

  const { adapter } = await getIam().auth.$context;
  return adapter.transaction(async (trx): Promise<AccountDeletionResult> => {
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

    await trx.delete({
      model: "user",
      where: [{ field: "id", value: userId }],
    });
    return { kind: "ok" };
  });
}
