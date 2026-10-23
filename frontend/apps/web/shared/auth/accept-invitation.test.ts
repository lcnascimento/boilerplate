import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { userIdSchema } from "./actor-ids";
import { getIam } from "./auth";
import { acceptInvitationFor } from "./invitation-acceptance";
import { ORGANIZATION_LIMIT } from "./policy";
import { fixtureDatabase } from "./test-database";

import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const ANA = {
  name: "Ana",
  email: "ana@invitations.test",
  password: "ana-password",
};
const DORA = {
  name: "Dora",
  email: "dora@invitations.test",
  password: "dora-password",
};
const UMBRELLA = "7a1d0c3e-0000-4000-8000-000001880000";
const UMBRELLA_INVITATION = "7a1d0c3e-0000-4000-8000-000001880100";
const DORA_ORGANIZATIONS = Array.from(
  { length: ORGANIZATION_LIMIT },
  (_, index) =>
    `7a1d0c3e-0000-4000-8000-0000018800${String(index + 1).padStart(2, "0")}`
);

const DAY_MS = 24 * 60 * 60 * 1000;

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "accepting an Invitation",
  () => {
    let database: Pool;
    let anaId: string;
    let doraId: string;

    async function removeFixtures() {
      await database.query(`DELETE FROM "user" WHERE email = ANY($1)`, [
        [ANA.email, DORA.email],
      ]);
      await database.query(`DELETE FROM organization WHERE id = ANY($1)`, [
        [UMBRELLA, ...DORA_ORGANIZATIONS],
      ]);
    }

    function acceptAsDora(headers: Headers) {
      return acceptInvitationFor(
        { userId: userIdSchema.parse(doraId), organizationId: null },
        headers,
        UMBRELLA_INVITATION
      );
    }

    async function sessionOf(user: typeof ANA): Promise<Headers> {
      const { headers } = await getIam().auth.api.signInEmail({
        body: { email: user.email, password: user.password },
        returnHeaders: true,
      });
      const cookie = headers
        .getSetCookie()
        .map((setCookie) => setCookie.split(";")[0])
        .join("; ");
      return new Headers({ cookie });
    }

    async function giveDoraOrganizations(count: number) {
      const ids = DORA_ORGANIZATIONS.slice(0, count);
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         SELECT id, 'Dora ' || id, 'invitations-' || id, now() FROM unnest($1::uuid[]) AS id`,
        [ids]
      );
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
         SELECT gen_random_uuid(), id, $2, 'viewer', now() FROM unnest($1::uuid[]) AS id`,
        [ids, doraId]
      );
    }

    async function inviteDora(expiresAt: Date) {
      await database.query(
        `INSERT INTO invitation (id, "organizationId", email, role, status, "expiresAt", "inviterId", "createdAt")
         VALUES ($1, $2, $3, 'admin', 'pending', $4, $5, now())`,
        [UMBRELLA_INVITATION, UMBRELLA, DORA.email, expiresAt, anaId]
      );
    }

    async function umbrellaInvitationStatus() {
      const { rows } = await database.query<{ status: string }>(
        `SELECT status FROM invitation WHERE id = $1`,
        [UMBRELLA_INVITATION]
      );
      return rows[0]?.status;
    }

    async function doraMemberships() {
      const { rows } = await database.query<{ name: string; role: string }>(
        `SELECT o.name, m.role FROM member m JOIN organization o ON o.id = m."organizationId"
          WHERE m."userId" = $1 AND o.id = $2`,
        [doraId, UMBRELLA]
      );
      return rows;
    }

    beforeAll(() => {
      database = fixtureDatabase();
    });

    beforeEach(async () => {
      await removeFixtures();
      const { auth } = getIam();
      anaId = (await auth.api.signUpEmail({ body: ANA })).user.id;
      doraId = (await auth.api.signUpEmail({ body: DORA })).user.id;
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Umbrella', 'invitations-umbrella', now())`,
        [UMBRELLA]
      );
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
         VALUES (gen_random_uuid(), $1, $2, 'owner', now())`,
        [UMBRELLA, anaId]
      );
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    it("makes the User a Member with the invited Role and the Organization active", async () => {
      await giveDoraOrganizations(ORGANIZATION_LIMIT - 1);
      await inviteDora(new Date(Date.now() + DAY_MS));
      const headers = await sessionOf(DORA);
      const { auth } = getIam();

      await acceptAsDora(headers);

      expect(await doraMemberships()).toEqual([
        { name: "Umbrella", role: "admin" },
      ]);
      expect(await umbrellaInvitationStatus()).toBe("accepted");
      const session = await auth.api.getSession({ headers });
      expect(session?.session.activeOrganizationId).toBe(UMBRELLA);
    });

    it("refuses the 11th Organization and leaves the Invitation pending", async () => {
      await giveDoraOrganizations(ORGANIZATION_LIMIT);
      await inviteDora(new Date(Date.now() + DAY_MS));

      await expect(acceptAsDora(await sessionOf(DORA))).rejects.toMatchObject({
        body: { code: "YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS" },
      });

      expect(await doraMemberships()).toEqual([]);
      expect(await umbrellaInvitationStatus()).toBe("pending");
    });

    it("refuses the 11th Organization when another one joins while accepting", async () => {
      await giveDoraOrganizations(ORGANIZATION_LIMIT - 1);
      await inviteDora(new Date(Date.now() + DAY_MS));
      const headers = await sessionOf(DORA);
      const joining = await database.connect();
      try {
        await joining.query("BEGIN");
        await joining.query(`SELECT id FROM "user" WHERE id = $1 FOR UPDATE`, [
          doraId,
        ]);
        await joining.query(
          `INSERT INTO organization (id, name, slug, "createdAt")
           VALUES ($1, 'Dora last', 'invitations-last', now())`,
          [DORA_ORGANIZATIONS[ORGANIZATION_LIMIT - 1]]
        );
        await joining.query(
          `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
           VALUES (gen_random_uuid(), $1, $2, 'viewer', now())`,
          [DORA_ORGANIZATIONS[ORGANIZATION_LIMIT - 1], doraId]
        );

        const acceptance = acceptAsDora(headers);
        await new Promise((resolve) => setTimeout(resolve, 300));
        await joining.query("COMMIT");

        await expect(acceptance).rejects.toMatchObject({
          body: {
            code: "YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS",
          },
        });
      } finally {
        joining.release();
      }
      expect(await doraMemberships()).toEqual([]);
      expect(await umbrellaInvitationStatus()).toBe("pending");
    });

    it("refuses an expired Invitation", async () => {
      await inviteDora(new Date(Date.now() - 1000));

      await expect(acceptAsDora(await sessionOf(DORA))).rejects.toMatchObject({
        body: { code: "INVITATION_NOT_FOUND" },
      });

      expect(await doraMemberships()).toEqual([]);
    });

    it("lets an Invitation live for 7 days", async () => {
      const before = Date.now();

      const invitation = await getIam().auth.api.createInvitation({
        headers: await sessionOf(ANA),
        body: { email: DORA.email, role: "viewer", organizationId: UMBRELLA },
      });

      const lifetime = new Date(invitation.expiresAt).getTime() - before;
      expect(Math.round(lifetime / DAY_MS)).toBe(7);
    });
  }
);
