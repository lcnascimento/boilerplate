import { trace } from "@opentelemetry/api";

import { prefixedActorIds, type ActorIds } from "./actor-ids";

export function annotateActorSpan(ids: ActorIds): void {
  const { userId, organizationId } = prefixedActorIds(ids);
  trace.getActiveSpan()?.setAttributes({
    "user.id": userId,
    "organization.id": organizationId,
  });
}
