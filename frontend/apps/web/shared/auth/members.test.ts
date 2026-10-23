import { describe, expect, it } from "vitest";

import { memberPolicy, ownershipPolicy, type Member } from "./members";

const ana: Member = {
  id: "m-ana",
  name: "Ana Owner",
  email: "owner@acme.test",
  role: "owner",
};
const bruno: Member = {
  id: "m-bruno",
  name: "Bruno Admin",
  email: "admin@acme.test",
  role: "admin",
};
const carla: Member = {
  id: "m-carla",
  name: "Carla Viewer",
  email: "viewer@acme.test",
  role: "viewer",
};

function actions(policy: ReturnType<typeof memberPolicy>) {
  return policy.rows.map(({ member, canChangeRole, canRemove }) => [
    member.email,
    { canChangeRole, canRemove },
  ]);
}

describe("memberPolicy", () => {
  it("lets the Owner manage everyone else", () => {
    const policy = memberPolicy(ana, [ana, bruno, carla]);

    expect(policy.canInvite).toBe(true);
    expect(policy.canCancelInvitations).toBe(true);
    expect(policy.assignableRoles).toEqual(["admin", "viewer"]);
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: true, canRemove: true }],
      ["viewer@acme.test", { canChangeRole: true, canRemove: true }],
    ]);
  });

  it("keeps an Admin away from the Owner and from the Owner Role", () => {
    const policy = memberPolicy(bruno, [ana, bruno, carla]);

    expect(policy.canInvite).toBe(true);
    expect(policy.canCancelInvitations).toBe(true);
    expect(policy.assignableRoles).toEqual(["admin", "viewer"]);
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: true, canRemove: false }],
      ["viewer@acme.test", { canChangeRole: true, canRemove: true }],
    ]);
  });

  it("gives a Viewer the list only", () => {
    const policy = memberPolicy(carla, [ana, bruno, carla]);

    expect(policy.canInvite).toBe(false);
    expect(policy.canCancelInvitations).toBe(false);
    expect(policy.assignableRoles).toEqual([]);
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: false, canRemove: false }],
      ["viewer@acme.test", { canChangeRole: false, canRemove: false }],
    ]);
  });
});

describe("ownershipPolicy", () => {
  it("lets the Owner hand the Organization to any other Member", () => {
    expect(ownershipPolicy(ana, [ana, bruno, carla])).toEqual({
      kind: "canTransfer",
      candidates: [bruno, carla],
    });
  });

  it("names the Owner to everyone else", () => {
    expect(ownershipPolicy(bruno, [ana, bruno, carla])).toEqual({
      kind: "ownedBy",
      owner: ana,
    });
    expect(ownershipPolicy(carla, [ana, bruno, carla])).toEqual({
      kind: "ownedBy",
      owner: ana,
    });
  });
});
