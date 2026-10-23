import { describe, expect, it } from "vitest";

import { pendingInvitations } from "./invitations";

const now = new Date("2026-10-22T12:00:00Z");
const nextWeek = new Date("2026-10-29T11:59:59Z");

describe("pendingInvitations", () => {
  it("keeps only the pending Invitations that have not expired", () => {
    const invitations = [
      { id: "pending", status: "pending", expiresAt: nextWeek },
      {
        id: "expired",
        status: "pending",
        expiresAt: new Date("2026-10-22T11:59:59Z"),
      },
      { id: "expiring now", status: "pending", expiresAt: now },
      { id: "accepted", status: "accepted", expiresAt: nextWeek },
      { id: "rejected", status: "rejected", expiresAt: nextWeek },
      { id: "canceled", status: "canceled", expiresAt: nextWeek },
    ];

    expect(pendingInvitations(invitations, now).map(({ id }) => id)).toEqual([
      "pending",
    ]);
  });
});
