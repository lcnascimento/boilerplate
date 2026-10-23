import { describe, expect, it } from "vitest";

import { pendingInvitations } from "./invitations";

const now = new Date("2026-10-22T12:00:00Z");
const nextWeek = new Date("2026-10-29T11:59:59Z");
const monday = new Date("2026-10-19T09:00:00Z");
const tuesday = new Date("2026-10-20T09:00:00Z");

describe("pendingInvitations", () => {
  it("keeps only the pending Invitations that have not expired, oldest first", () => {
    const invitations = [
      {
        id: "pending since tuesday",
        status: "pending",
        createdAt: tuesday,
        expiresAt: nextWeek,
      },
      {
        id: "expired",
        status: "pending",
        createdAt: monday,
        expiresAt: new Date("2026-10-22T11:59:59Z"),
      },
      {
        id: "expiring now",
        status: "pending",
        createdAt: monday,
        expiresAt: now,
      },
      {
        id: "pending since monday",
        status: "pending",
        createdAt: monday,
        expiresAt: nextWeek,
      },
      {
        id: "accepted",
        status: "accepted",
        createdAt: monday,
        expiresAt: nextWeek,
      },
      {
        id: "rejected",
        status: "rejected",
        createdAt: monday,
        expiresAt: nextWeek,
      },
      {
        id: "canceled",
        status: "canceled",
        createdAt: monday,
        expiresAt: nextWeek,
      },
    ];

    expect(pendingInvitations(invitations, now).map(({ id }) => id)).toEqual([
      "pending since monday",
      "pending since tuesday",
    ]);
  });
});
