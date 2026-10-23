import "server-only";

import { runWithTransaction } from "@better-auth/core/context";

import { runAsActor, type AuditedActor } from "./audit-connection";
import { getIam } from "./auth";

export async function acceptInvitationFor(
  acceptor: AuditedActor,
  headers: Headers,
  invitationId: string
) {
  const { auth } = getIam();
  const { adapter } = await auth.$context;
  return runAsActor(acceptor, headers, () =>
    runWithTransaction(adapter, () =>
      auth.api.acceptInvitation({ headers, body: { invitationId } })
    )
  );
}
