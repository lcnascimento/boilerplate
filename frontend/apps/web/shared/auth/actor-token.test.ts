import { createLocalJWKSet, decodeProtectedHeader, jwtVerify } from "jose";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { findActor, type Actor } from "./actor";
import { parseActorIds, type ActorIds } from "./actor-ids";
import { mintActorToken } from "./actor-token";
import { getIam } from "./auth";

vi.mock("server-only", () => ({}));

const UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "mintActorToken",
  () => {
    let acmeOwner: Actor;
    let globexOwnerInAcme: ActorIds;

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

      acmeOwner = await findActor(getIam().database, {
        userId: await userId("owner@acme.test"),
        organizationId: acme.id,
      });
      globexOwnerInAcme = parseActorIds({
        userId: await userId("owner@globex.test"),
        organizationId: acme.id,
      });
    });

    afterAll(async () => {
      await getIam().database.destroy();
    });

    async function verify(token: string) {
      const jwks = createLocalJWKSet(await getIam().auth.api.getJwks());
      return jwtVerify(token, jwks, {
        issuer: process.env.BETTER_AUTH_URL,
        audience: "boilerplate-api",
        algorithms: ["EdDSA"],
      });
    }

    it("signs the Actor's prefixed ids and Role, verifiable by the JWKS, for 5 minutes", async () => {
      const token = await mintActorToken(acmeOwner);

      const { payload, protectedHeader } = await verify(token);

      expect(protectedHeader.alg).toBe("EdDSA");
      expect(protectedHeader.kid).toMatch(UUID);
      expect(payload).toMatchObject({
        sub: `user_${acmeOwner.userId}`,
        org: `org_${acmeOwner.organizationId}`,
        role: "owner",
        iss: process.env.BETTER_AUTH_URL,
        aud: "boilerplate-api",
      });
      expect((payload.exp ?? 0) - (payload.iat ?? 0)).toBe(300);
    });

    it("signs for the given TTL", async () => {
      const token = await mintActorToken(acmeOwner, 3600);

      const { payload } = await verify(token);

      expect((payload.exp ?? 0) - (payload.iat ?? 0)).toBe(3600);
      expect(decodeProtectedHeader(token).alg).toBe("EdDSA");
    });

    it("finds no Actor for a User who is not a Member of the Organization", async () => {
      await expect(
        findActor(getIam().database, globexOwnerInAcme)
      ).rejects.toThrow(
        `user ${globexOwnerInAcme.userId} is not a member of organization ${globexOwnerInAcme.organizationId}`
      );
    });
  }
);
