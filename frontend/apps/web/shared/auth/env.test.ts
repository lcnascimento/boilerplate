import { describe, expect, it } from "vitest";

import { parseIamEnv } from "./env";

const valid = {
  IAM_DATABASE_URL: "postgres://iam:iam@localhost:5432/iam?sslmode=disable",
  BETTER_AUTH_SECRET: "a-dev-secret-that-is-long-enough-32",
  BETTER_AUTH_URL: "http://localhost:3000",
};

describe("parseIamEnv", () => {
  it("accepts the dev-only secret outside production", () => {
    expect(
      parseIamEnv({
        ...valid,
        NODE_ENV: "development",
        BETTER_AUTH_SECRET: "dev-only-secret-change-me-in-production",
      }).BETTER_AUTH_SECRET
    ).toBe("dev-only-secret-change-me-in-production");
  });

  it("reads the database, the secret and the URL, without a trailing slash", () => {
    expect(
      parseIamEnv({
        ...valid,
        BETTER_AUTH_URL: "http://localhost:3000/",
        PATH: "/bin",
      })
    ).toEqual({
      IAM_DATABASE_URL: "postgres://iam:iam@localhost:5432/iam?sslmode=disable",
      BETTER_AUTH_SECRET: "a-dev-secret-that-is-long-enough-32",
      BETTER_AUTH_URL: "http://localhost:3000",
    });
  });

  it.each([
    [
      {},
      "Invalid environment: IAM_DATABASE_URL is required; BETTER_AUTH_SECRET is required; BETTER_AUTH_URL is required",
    ],
    [
      { ...valid, IAM_DATABASE_URL: "http://localhost:5432/iam" },
      "Invalid environment: IAM_DATABASE_URL must be a postgres:// URL",
    ],
    [
      { ...valid, BETTER_AUTH_SECRET: "short" },
      "Invalid environment: BETTER_AUTH_SECRET must have at least 32 characters",
    ],
    [
      {
        ...valid,
        NODE_ENV: "production",
        BETTER_AUTH_SECRET: "dev-only-secret-change-me-in-production",
      },
      "Invalid environment: BETTER_AUTH_SECRET is the dev-only secret; generate one for production (openssl rand -base64 32)",
    ],
    [
      { ...valid, BETTER_AUTH_URL: "" },
      "Invalid environment: BETTER_AUTH_URL must be an http(s) URL",
    ],
    [
      { ...valid, BETTER_AUTH_URL: "localhost:3000" },
      "Invalid environment: BETTER_AUTH_URL must be an http(s) URL",
    ],
  ])("rejects %j with a clear message", (env, message) => {
    expect(() => parseIamEnv(env)).toThrow(message);
  });
});
