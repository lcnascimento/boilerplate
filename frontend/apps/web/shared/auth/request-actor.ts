import "server-only";

import { cache } from "react";

import { actorOf, listMembersOldestFirst, type Actor } from "./actor";
import { annotateActorSpan } from "./actor-span";
import { getIam } from "./auth";
import { getRequestSession } from "./session";

import type { UserOrganizations } from "./organizations";

export type RequestActor =
  | { kind: "signed-out" }
  | { kind: "no-organization" }
  | { kind: "actor"; actor: Actor; organizations: UserOrganizations };

interface RequestSession {
  token: string;
  userId: string;
  activeOrganizationId?: string | null;
}

export async function resolveRequestActor({
  token,
  userId,
  activeOrganizationId = null,
}: RequestSession): Promise<Exclude<RequestActor, { kind: "signed-out" }>> {
  const { auth, database } = getIam();
  const members = await listMembersOldestFirst(database, userId);
  const active =
    members.find(
      ({ organization }) => organization.id === activeOrganizationId
    ) ?? members[0];

  const resolvedId = active?.organization.id ?? null;
  if (resolvedId !== activeOrganizationId) {
    const { internalAdapter } = await auth.$context;
    await internalAdapter.updateSession(token, {
      activeOrganizationId: resolvedId,
    });
  }

  if (active === undefined) return { kind: "no-organization" };
  return {
    kind: "actor",
    actor: actorOf(userId, active),
    organizations: {
      active: active.organization,
      organizations: members.map(({ organization }) => organization),
    },
  };
}

const cachedRequestActor = cache(async (): Promise<RequestActor> => {
  const session = await getRequestSession();
  if (session === null) return { kind: "signed-out" };
  return resolveRequestActor(session.session);
});

export async function getRequestActor(): Promise<RequestActor> {
  const requestActor = await cachedRequestActor();
  if (requestActor.kind === "actor") annotateActorSpan(requestActor.actor);
  return requestActor;
}
