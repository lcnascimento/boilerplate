import { createLocalJWKSet, decodeProtectedHeader, jwtVerify } from "jose";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { mintActorToken } from "./actor-token";
import { getIam } from "./auth";

vi.mock("server-only", () => ({}));

const UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

// Runs against the iam database of the Tilt stack, seeded by iam-seed: ci-e2e sets IAM_DATABASE_URL for it.
describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "mintActorToken",
  () => {
    let acmeId = "";
    let acmeOwnerId = "";
    let globexOwnerId = "";

    beforeAll(async () => {
      const context = await getIam().auth.$context;
      const userId = async (email: string) => {
        const found = await context.internalAdapter.findUserByEmail(email);
        if (found === null) throw new Error(`seed user ${email} is missing`);
        return found.user.id;
      };
      const acme = await context.adapter.findOne<{ id: string }>({
        model: "organization",
        where: [{ field: "slug", value: "acme" }],
      });
      if (acme === null) throw new Error("seed organization acme is missing");

      acmeId = acme.id;
      acmeOwnerId = await userId("owner@acme.test");
      globexOwnerId = await userId("owner@globex.test");
    });

    afterAll(async () => {
      await getIam().database.end();
    });

    async function verify(token: string) {
      const jwks = createLocalJWKSet(await getIam().auth.api.getJwks());
      return jwtVerify(token, jwks, {
        issuer: process.env.BETTER_AUTH_URL,
        audience: "boilerplate-api",
        algorithms: ["EdDSA"],
      });
    }

    it("signs the member's prefixed ids and Role, verifiable by the JWKS, for 5 minutes", async () => {
      const token = await mintActorToken({
        userId: acmeOwnerId,
        organizationId: acmeId,
      });

      const { payload, protectedHeader } = await verify(token);

      expect(protectedHeader.alg).toBe("EdDSA");
      expect(protectedHeader.kid).toMatch(UUID);
      expect(payload).toMatchObject({
        sub: `user_${acmeOwnerId}`,
        org: `org_${acmeId}`,
        role: "owner",
        iss: process.env.BETTER_AUTH_URL,
        aud: "boilerplate-api",
      });
      expect(acmeOwnerId).toMatch(UUID);
      expect(acmeId).toMatch(UUID);
      expect((payload.exp ?? 0) - (payload.iat ?? 0)).toBe(300);
    });

    it("signs for the given TTL", async () => {
      const token = await mintActorToken({
        userId: acmeOwnerId,
        organizationId: acmeId,
        ttlSeconds: 3600,
      });

      const { payload } = await verify(token);

      expect((payload.exp ?? 0) - (payload.iat ?? 0)).toBe(3600);
      expect(decodeProtectedHeader(token).alg).toBe("EdDSA");
    });

    it("refuses a User who is not a Member of the Organization", async () => {
      await expect(
        mintActorToken({ userId: globexOwnerId, organizationId: acmeId })
      ).rejects.toMatchObject({
        name: "ActorTokenError",
        reason: "not_a_member",
      });
    });

    it.each([["member"], ["owner,admin"], [""]])(
      "refuses a Member whose Role is %j",
      async (role) => {
        const { adapter } = await getIam().auth.$context;
        vi.spyOn(adapter, "findOne").mockResolvedValueOnce({ role });

        await expect(
          mintActorToken({ userId: acmeOwnerId, organizationId: acmeId })
        ).rejects.toMatchObject({
          name: "ActorTokenError",
          reason: "unknown_role",
        });
      }
    );
  }
);
