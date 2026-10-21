import { describe, expect, it } from "vitest";

import { memberPolicy, type Member } from "./members";

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
const diego: Member = {
  id: "m-diego",
  name: "Diego Owner",
  email: "owner@globex.test",
  role: "owner",
};

function actions(policy: ReturnType<typeof memberPolicy>) {
  return policy.rows.map(({ member, canChangeRole, canRemove }) => [
    member.email,
    { canChangeRole, canRemove },
  ]);
}

describe("memberPolicy", () => {
  it("lets the sole Owner manage everyone else but not leave", () => {
    const policy = memberPolicy(ana, [ana, bruno, carla]);

    expect(policy.canAdd).toBe(true);
    expect(policy.assignableRoles).toEqual(["owner", "admin", "viewer"]);
    expect(policy.leave).toBe("lastOwner");
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: true, canRemove: true }],
      ["viewer@acme.test", { canChangeRole: true, canRemove: true }],
    ]);
  });

  it("lets an Owner demote themselves and leave while another Owner remains", () => {
    const policy = memberPolicy(ana, [ana, diego, carla]);

    expect(policy.leave).toBe("allowed");
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: true, canRemove: false }],
      ["owner@globex.test", { canChangeRole: true, canRemove: true }],
      ["viewer@acme.test", { canChangeRole: true, canRemove: true }],
    ]);
  });

  it("keeps an Admin away from Owners and from the Owner Role", () => {
    const policy = memberPolicy(bruno, [ana, bruno, carla]);

    expect(policy.canAdd).toBe(true);
    expect(policy.assignableRoles).toEqual(["admin", "viewer"]);
    expect(policy.leave).toBe("allowed");
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: true, canRemove: false }],
      ["viewer@acme.test", { canChangeRole: true, canRemove: true }],
    ]);
  });

  it("gives a Viewer the list and the way out only", () => {
    const policy = memberPolicy(carla, [ana, bruno, carla]);

    expect(policy.canAdd).toBe(false);
    expect(policy.assignableRoles).toEqual([]);
    expect(policy.leave).toBe("allowed");
    expect(actions(policy)).toEqual([
      ["owner@acme.test", { canChangeRole: false, canRemove: false }],
      ["admin@acme.test", { canChangeRole: false, canRemove: false }],
      ["viewer@acme.test", { canChangeRole: false, canRemove: false }],
    ]);
  });
});
