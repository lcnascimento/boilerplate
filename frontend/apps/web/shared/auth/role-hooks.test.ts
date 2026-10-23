import { APIError } from "better-auth/api";
import { describe, expect, it } from "vitest";

import { roleHooks } from "./role-hooks";

const organizationWithMembers = "org-acme";
const newOrganization = "org-new";

const hooks = roleHooks((organizationId) =>
  Promise.resolve(organizationId === organizationWithMembers)
);

async function codeOf(attempt: Promise<unknown>): Promise<string | undefined> {
  try {
    await attempt;
    return undefined;
  } catch (error) {
    if (error instanceof APIError) return String(error.body?.code);
    throw error;
  }
}

function member(organizationId: string, role: string) {
  return { member: { userId: "user-ana", organizationId, role } };
}

describe("roleHooks.beforeAddMember", () => {
  it("makes the creator of a new Organization its Owner", async () => {
    expect(
      await codeOf(hooks.beforeAddMember(member(newOrganization, "owner")))
    ).toBeUndefined();
  });

  it("adds Admins and Viewers to an existing Organization", async () => {
    for (const role of ["admin", "viewer"]) {
      expect(
        await codeOf(
          hooks.beforeAddMember(member(organizationWithMembers, role))
        )
      ).toBeUndefined();
    }
  });

  it("refuses a second Owner", async () => {
    expect(
      await codeOf(
        hooks.beforeAddMember(member(organizationWithMembers, "owner"))
      )
    ).toBe("OWNER_NOT_ASSIGNABLE");
    expect(
      await codeOf(
        hooks.beforeAddMember(member(organizationWithMembers, "admin,owner"))
      )
    ).toBe("OWNER_NOT_ASSIGNABLE");
  });

  it("refuses an unknown Role", async () => {
    expect(
      await codeOf(
        hooks.beforeAddMember(member(organizationWithMembers, "member"))
      )
    ).toBe("UNKNOWN_ROLE");
  });
});

describe("roleHooks.beforeUpdateMemberRole", () => {
  function update(currentRole: string, newRole: string) {
    return hooks.beforeUpdateMemberRole({
      member: { role: currentRole },
      newRole,
    });
  }

  it("moves a Member between Admin and Viewer", async () => {
    expect(await codeOf(update("admin", "viewer"))).toBeUndefined();
    expect(await codeOf(update("viewer", "admin"))).toBeUndefined();
  });

  it("never makes an Owner", async () => {
    expect(await codeOf(update("admin", "owner"))).toBe("OWNER_NOT_ASSIGNABLE");
  });

  it("keeps the Owner until the ownership is transferred", async () => {
    expect(await codeOf(update("owner", "admin"))).toBe("OWNER_MUST_TRANSFER");
  });

  it("refuses an unknown Role", async () => {
    expect(await codeOf(update("viewer", "superadmin"))).toBe("UNKNOWN_ROLE");
  });
});

describe("roleHooks.beforeRemoveMember", () => {
  it("removes an Admin or a Viewer", async () => {
    for (const role of ["admin", "viewer"]) {
      expect(
        await codeOf(hooks.beforeRemoveMember({ member: { role } }))
      ).toBeUndefined();
    }
  });

  it("keeps the Owner until the ownership is transferred", async () => {
    expect(
      await codeOf(hooks.beforeRemoveMember({ member: { role: "owner" } }))
    ).toBe("OWNER_MUST_TRANSFER");
  });
});

describe("roleHooks.beforeCreateInvitation", () => {
  it("invites Admins and Viewers only", async () => {
    expect(
      await codeOf(
        hooks.beforeCreateInvitation({ invitation: { role: "viewer" } })
      )
    ).toBeUndefined();
    expect(
      await codeOf(
        hooks.beforeCreateInvitation({ invitation: { role: "owner" } })
      )
    ).toBe("OWNER_NOT_ASSIGNABLE");
  });
});
