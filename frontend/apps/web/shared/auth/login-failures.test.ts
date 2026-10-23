import { betterAuth } from "better-auth";
import { memoryAdapter } from "better-auth/adapters/memory";
import { beforeEach, describe, expect, it } from "vitest";

import {
  LOGIN_FAILURES_METRIC,
  loginFailureHook,
  seedLoginFailures,
  withLoginFailures,
} from "./login-failures";
import { startTestTelemetry } from "./test-telemetry";

const telemetry = startTestTelemetry();

const BASE_URL = "http://localhost:3000";

function createAuth() {
  return betterAuth({
    baseURL: BASE_URL,
    secret: "a-test-secret-that-is-at-least-32-characters-long",
    database: memoryAdapter({
      user: [],
      session: [],
      account: [],
      verification: [],
      rateLimit: [],
    }),
    emailAndPassword: { enabled: true, minPasswordLength: 8 },
    rateLimit: { enabled: true, storage: "memory" },
    hooks: { after: loginFailureHook },
  });
}

type Auth = ReturnType<typeof createAuth>;

const ana = { email: "ana@acme.test", password: "correct-horse", name: "Ana" };

async function failures(): Promise<Record<string, number>> {
  return telemetry.counter(LOGIN_FAILURES_METRIC);
}

async function rejects(call: Promise<unknown>): Promise<void> {
  await expect(call).rejects.toThrow();
}

function post(path: string, body: unknown): Request {
  return new Request(`${BASE_URL}/api/auth${path}`, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      origin: BASE_URL,
      "x-forwarded-for": "203.0.113.7",
    },
    body: JSON.stringify(body),
  });
}

describe("seedLoginFailures", () => {
  it("creates every reason at zero, so the series exist before the first failure", async () => {
    seedLoginFailures();

    expect(Object.keys(await failures()).sort()).toEqual([
      "reason=email_taken",
      "reason=invalid_credentials",
      "reason=invalid_input",
      "reason=rate_limited",
    ]);
  });
});

describe("login failures", () => {
  let auth: Auth;

  beforeEach(async () => {
    auth = createAuth();
    await auth.api.signUpEmail({ body: ana });
  });

  it("counts a wrong password as invalid_credentials", async () => {
    const before = await failures();

    await rejects(
      auth.api.signInEmail({
        body: { email: ana.email, password: "wrong-password" },
      })
    );

    const after = await failures();
    expect(
      (after["reason=invalid_credentials"] ?? 0) -
        (before["reason=invalid_credentials"] ?? 0)
    ).toBe(1);
  });

  it("counts a sign-up with a registered email as email_taken", async () => {
    const before = await failures();

    await rejects(auth.api.signUpEmail({ body: ana }));

    const after = await failures();
    expect(
      (after["reason=email_taken"] ?? 0) - (before["reason=email_taken"] ?? 0)
    ).toBe(1);
  });

  it("counts a malformed email or a short password as invalid_input", async () => {
    const before = await failures();

    await rejects(
      auth.api.signInEmail({
        body: { email: "not-an-email", password: "whatever-it-is" },
      })
    );
    await rejects(
      auth.api.signUpEmail({
        body: { email: "bruno@acme.test", password: "short", name: "Bruno" },
      })
    );

    const after = await failures();
    expect(
      (after["reason=invalid_input"] ?? 0) -
        (before["reason=invalid_input"] ?? 0)
    ).toBe(2);
  });

  it("does not count a successful sign-in, nor failures outside sign-in and sign-up", async () => {
    const before = await failures();

    await auth.api.signInEmail({
      body: { email: ana.email, password: ana.password },
    });
    await rejects(
      auth.api.changePassword({
        body: {
          currentPassword: "wrong-password",
          newPassword: "another-horse",
        },
        headers: new Headers(),
      })
    );

    expect(await failures()).toEqual(before);
  });

  it("counts the sign-ins the rate limiter refuses as rate_limited", async () => {
    const handle = withLoginFailures((request) => auth.handler(request));
    const before = await failures();

    const statuses: number[] = [];
    for (let attempt = 0; attempt < 4; attempt++) {
      const response = await handle(
        post("/sign-in/email", { email: ana.email, password: "wrong-password" })
      );
      statuses.push(response.status);
    }

    const after = await failures();
    expect(statuses).toEqual([401, 401, 401, 429]);
    expect(
      (after["reason=rate_limited"] ?? 0) - (before["reason=rate_limited"] ?? 0)
    ).toBe(1);
    expect(
      (after["reason=invalid_credentials"] ?? 0) -
        (before["reason=invalid_credentials"] ?? 0)
    ).toBe(3);
  });

  it("never puts the email in a label", async () => {
    await rejects(
      auth.api.signInEmail({
        body: { email: ana.email, password: "wrong-password" },
      })
    );

    expect(Object.keys(await failures()).join(" ")).not.toContain(ana.email);
  });
});
