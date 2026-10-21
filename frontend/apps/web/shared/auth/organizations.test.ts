import { describe, expect, it } from "vitest";

import { pickActiveOrganization, type Organization } from "./organizations";

const acme: Organization = { id: "org-acme", name: "Acme" };
const globex: Organization = { id: "org-globex", name: "Globex" };

describe("pickActiveOrganization", () => {
  it("keeps the active Organization while the User is still its Member", () => {
    expect(pickActiveOrganization([acme, globex], "org-globex")).toEqual(
      globex
    );
  });

  it("falls back to the oldest membership once the active one is gone", () => {
    expect(pickActiveOrganization([acme, globex], "org-initech")).toEqual(acme);
  });

  it("falls back to the oldest membership when none is active", () => {
    expect(pickActiveOrganization([globex, acme], null)).toEqual(globex);
  });

  it("has nothing to pick for a User without memberships", () => {
    expect(pickActiveOrganization([], "org-acme")).toBeNull();
  });
});
