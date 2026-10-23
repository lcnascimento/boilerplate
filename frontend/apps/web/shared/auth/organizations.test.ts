import { describe, expect, it } from "vitest";

import {
  organizationNameInputSchema,
  organizationSettingsPolicy,
} from "./organizations";

describe("organizationSettingsPolicy", () => {
  it("lets the Owner rename and delete, and asks for a transfer before leaving", () => {
    expect(organizationSettingsPolicy("owner")).toEqual({
      canRename: true,
      canDelete: true,
      leave: "ownerMustTransfer",
    });
  });

  it("lets an Admin rename and leave, but not delete", () => {
    expect(organizationSettingsPolicy("admin")).toEqual({
      canRename: true,
      canDelete: false,
      leave: "allowed",
    });
  });

  it("lets a Viewer only leave", () => {
    expect(organizationSettingsPolicy("viewer")).toEqual({
      canRename: false,
      canDelete: false,
      leave: "allowed",
    });
  });
});

describe("organizationNameInputSchema", () => {
  it("trims the name", () => {
    expect(organizationNameInputSchema.parse({ name: "  Initech  " })).toEqual({
      name: "Initech",
    });
  });

  it("refuses a blank or too long name", () => {
    expect(organizationNameInputSchema.safeParse({ name: "   " }).success).toBe(
      false
    );
    expect(
      organizationNameInputSchema.safeParse({ name: "x".repeat(101) }).success
    ).toBe(false);
  });
});
