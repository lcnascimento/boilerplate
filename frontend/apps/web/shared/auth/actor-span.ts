import { trace } from "@opentelemetry/api";

interface ActorIds {
  userId: string;
  organizationId: string;
}

// The ids only: email and name never reach telemetry.
export function annotateActorSpan({ userId, organizationId }: ActorIds): void {
  trace.getActiveSpan()?.setAttributes({
    "user.id": userId,
    "organization.id": organizationId,
  });
}
