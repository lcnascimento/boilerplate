import { trace } from "@opentelemetry/api";
import { describe, expect, it } from "vitest";

import { parseActorIds } from "./actor-ids";
import { annotateActorSpan } from "./actor-span";
import { startTestTelemetry } from "./test-telemetry";

const telemetry = startTestTelemetry();

describe("annotateActorSpan", () => {
  it("puts the prefixed user and organization ids on the active span", async () => {
    trace.getTracer("test").startActiveSpan("POST /notes", (span) => {
      annotateActorSpan(
        parseActorIds({
          userId: "5d2a7c10-3e4f-4b6a-8c9d-0e1f2a3b4c5d",
          organizationId: "9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
        })
      );
      span.end();
    });

    const [span] = await telemetry.spans();
    expect(span?.attributes).toEqual({
      "user.id": "user_5d2a7c10-3e4f-4b6a-8c9d-0e1f2a3b4c5d",
      "organization.id": "org_9b2e4c71-1a3f-4d5e-8f60-7a8b9c0d1e2f",
    });
  });
});
