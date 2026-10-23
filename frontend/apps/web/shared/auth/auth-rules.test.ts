import { betterAuth } from "better-auth";
import { memoryAdapter } from "better-auth/adapters/memory";
import { APIError } from "better-auth/api";
import { organization } from "better-auth/plugins";
import { describe, expect, it } from "vitest";

import { authRulesHook, type AuthRule } from "./auth-rules";

function createAuth(rules: readonly AuthRule[]) {
  return betterAuth({
    baseURL: "http://localhost:3000",
    secret: "a-test-secret-that-is-at-least-32-characters-long",
    database: memoryAdapter({
      user: [],
      session: [],
      account: [],
      verification: [],
      organization: [],
      member: [],
      invitation: [],
    }),
    emailAndPassword: { enabled: true },
    hooks: { before: authRulesHook(rules) },
    plugins: [organization()],
  });
}

const ana = { email: "ana@acme.test", password: "correct-horse", name: "Ana" };

function refuse(kind: string): AuthRule {
  return (action) => {
    if (action.kind !== kind) return Promise.resolve();
    return Promise.reject(
      new APIError("FORBIDDEN", { code: "BLOCKED", message: "Blocked" })
    );
  };
}

describe("authRulesHook", () => {
  it("lets every action through when no rule refuses it", async () => {
    const auth = createAuth([refuse("nothing")]);

    const { user } = await auth.api.signUpEmail({ body: ana });
    await auth.api.signInEmail({ body: ana });
    const created = await auth.api.createOrganization({
      body: { name: "Acme", slug: "acme", userId: user.id },
    });

    expect(created).toMatchObject({ name: "Acme" });
  });

  it("refuses a sign-up with the rule's error", async () => {
    const auth = createAuth([refuse("signUp")]);

    await expect(auth.api.signUpEmail({ body: ana })).rejects.toMatchObject({
      body: { code: "BLOCKED" },
    });
  });

  it("refuses a sign-in with the rule's error", async () => {
    const auth = createAuth([refuse("signIn")]);
    await auth.api.signUpEmail({ body: ana });

    await expect(auth.api.signInEmail({ body: ana })).rejects.toMatchObject({
      body: { code: "BLOCKED" },
    });
  });

  it("refuses creating an Organization with the rule's error", async () => {
    const auth = createAuth([refuse("createOrganization")]);
    const { user } = await auth.api.signUpEmail({ body: ana });

    await expect(
      auth.api.createOrganization({
        body: { name: "Acme", slug: "acme", userId: user.id },
      })
    ).rejects.toMatchObject({ body: { code: "BLOCKED" } });
  });

  it("refuses an Invitation with the rule's error and hands it the invited email", async () => {
    const seen: unknown[] = [];
    const auth = createAuth([
      (action) => {
        if (action.kind !== "createInvitation") return Promise.resolve();
        seen.push(action);
        return refuse("createInvitation")(action);
      },
    ]);
    const { user } = await auth.api.signUpEmail({ body: ana });
    const { id: organizationId } = await auth.api.createOrganization({
      body: { name: "Acme", slug: "acme", userId: user.id },
    });
    const { headers } = await auth.api.signInEmail({
      body: ana,
      returnHeaders: true,
    });
    const cookie = headers.get("set-cookie")?.split(";")[0] ?? "";

    await expect(
      auth.api.createInvitation({
        headers: new Headers({ cookie }),
        body: { email: "bruno@acme.test", role: "member", organizationId },
      })
    ).rejects.toMatchObject({ body: { code: "BLOCKED" } });
    expect(seen).toEqual([
      { kind: "createInvitation", email: "bruno@acme.test" },
    ]);
  });

  it("hands the rule the email a sign-up uses", async () => {
    const seen: unknown[] = [];
    const auth = createAuth([
      (action) => {
        seen.push(action);
        return Promise.resolve();
      },
    ]);

    await auth.api.signUpEmail({ body: ana });

    expect(seen).toEqual([{ kind: "signUp", email: "ana@acme.test" }]);
  });
});
