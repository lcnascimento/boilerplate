import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { getIam } from "./auth";
import { resolveRequestActor } from "./request-actor";

import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const USER = "7a1d0c3e-0000-4000-8000-000000000001";
const INITECH = "7a1d0c3e-0000-4000-8000-0000000000a1";
const UMBRELLA = "7a1d0c3e-0000-4000-8000-0000000000a2";
const HOOLI = "7a1d0c3e-0000-4000-8000-0000000000a3";
const INITECH_MEMBER = "7a1d0c3e-0000-4000-8000-0000000000b1";
const UMBRELLA_MEMBER = "7a1d0c3e-0000-4000-8000-0000000000b2";
const HOOLI_MEMBER = "7a1d0c3e-0000-4000-8000-0000000000b3";
const TOKEN = "request-actor-test-session";

const JANUARY = "2026-01-01T00:00:00Z";
const FEBRUARY = "2026-02-01T00:00:00Z";

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "resolveRequestActor",
  () => {
    let database: Pool;

    beforeAll(() => {
      database = getIam().database;
    });

    async function removeFixtures() {
      await database.query(`DELETE FROM "user" WHERE id = $1`, [USER]);
      await database.query(`DELETE FROM organization WHERE id = ANY($1)`, [
        [INITECH, UMBRELLA, HOOLI],
      ]);
    }

    beforeEach(async () => {
      await removeFixtures();
      await database.query(
        `INSERT INTO "user" (id, name, email, "emailVerified") VALUES ($1, 'Rita', 'rita@request-actor.test', true)`,
        [USER]
      );
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Hooli', 'request-actor-hooli', now()),
                ($2, 'Umbrella', 'request-actor-umbrella', now()),
                ($3, 'Initech', 'request-actor-initech', now())`,
        [HOOLI, UMBRELLA, INITECH]
      );
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    async function addMember(
      id: string,
      organizationId: string,
      role: string,
      createdAt: string
    ) {
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt") VALUES ($1, $2, $3, $4, $5)`,
        [id, organizationId, USER, role, createdAt]
      );
    }

    async function openSession(activeOrganizationId: string | null) {
      await database.query(
        `INSERT INTO session (token, "userId", "activeOrganizationId", "expiresAt", "updatedAt")
         VALUES ($1, $2, $3, now() + interval '1 hour', now())`,
        [TOKEN, USER, activeOrganizationId]
      );
      return { token: TOKEN, userId: USER, activeOrganizationId };
    }

    async function storedActiveOrganizationId() {
      const { rows } = await database.query<{
        activeOrganizationId: string | null;
      }>(`SELECT "activeOrganizationId" FROM session WHERE token = $1`, [
        TOKEN,
      ]);
      return rows[0]?.activeOrganizationId;
    }

    it("keeps the active Organization, with the Member's Role and every Organization oldest membership first", async () => {
      await addMember(UMBRELLA_MEMBER, UMBRELLA, "viewer", FEBRUARY);
      await addMember(INITECH_MEMBER, INITECH, "owner", JANUARY);

      const requestActor = await resolveRequestActor(
        await openSession(UMBRELLA)
      );

      expect(requestActor).toEqual({
        kind: "actor",
        actor: {
          userId: USER,
          organizationId: UMBRELLA,
          memberId: UMBRELLA_MEMBER,
          role: "viewer",
        },
        organizations: {
          active: { id: UMBRELLA, name: "Umbrella" },
          organizations: [
            { id: INITECH, name: "Initech" },
            { id: UMBRELLA, name: "Umbrella" },
          ],
        },
      });
      expect(await storedActiveOrganizationId()).toBe(UMBRELLA);
    });

    it("activates the oldest membership for a session without an active Organization", async () => {
      await addMember(UMBRELLA_MEMBER, UMBRELLA, "viewer", FEBRUARY);
      await addMember(INITECH_MEMBER, INITECH, "admin", JANUARY);

      const requestActor = await resolveRequestActor(await openSession(null));

      expect(requestActor).toMatchObject({
        kind: "actor",
        actor: {
          organizationId: INITECH,
          memberId: INITECH_MEMBER,
          role: "admin",
        },
      });
      expect(await storedActiveOrganizationId()).toBe(INITECH);
    });

    it("moves a User removed from the active Organization to the oldest remaining membership", async () => {
      await addMember(HOOLI_MEMBER, HOOLI, "owner", JANUARY);
      await addMember(UMBRELLA_MEMBER, UMBRELLA, "admin", FEBRUARY);
      const session = await openSession(HOOLI);
      await database.query(`DELETE FROM member WHERE id = $1`, [HOOLI_MEMBER]);

      const requestActor = await resolveRequestActor(session);

      expect(requestActor).toMatchObject({
        kind: "actor",
        actor: { organizationId: UMBRELLA, role: "admin" },
        organizations: {
          organizations: [{ id: UMBRELLA, name: "Umbrella" }],
        },
      });
      expect(await storedActiveOrganizationId()).toBe(UMBRELLA);
    });

    it("breaks a tie between memberships created at the same instant by Organization id", async () => {
      await addMember(UMBRELLA_MEMBER, UMBRELLA, "viewer", JANUARY);
      await addMember(INITECH_MEMBER, INITECH, "owner", JANUARY);

      const requestActor = await resolveRequestActor(await openSession(null));

      expect(requestActor).toMatchObject({
        kind: "actor",
        actor: { organizationId: INITECH, role: "owner" },
        organizations: {
          organizations: [
            { id: INITECH, name: "Initech" },
            { id: UMBRELLA, name: "Umbrella" },
          ],
        },
      });
    });

    it("finds no Organization for a User without memberships, and clears the stale active one", async () => {
      const requestActor = await resolveRequestActor(
        await openSession(INITECH)
      );

      expect(requestActor).toEqual({ kind: "no-organization" });
      expect(await storedActiveOrganizationId()).toBeNull();
    });
  }
);
