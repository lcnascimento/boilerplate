import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { organizationIdSchema, userIdSchema } from "./actor-ids";
import { deleteOrganization } from "./organization-deletion";
import { fixtureDatabase } from "./test-database";

import type { Actor } from "./actor";
import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const ANA = "7a1d0c3e-0000-4000-8000-000000000201";
const BRUNO = "7a1d0c3e-0000-4000-8000-000000000202";
const CARLA = "7a1d0c3e-0000-4000-8000-000000000203";
const INITECH = "7a1d0c3e-0000-4000-8000-0000000002a1";
const HOOLI = "7a1d0c3e-0000-4000-8000-0000000002a2";
const ANA_MEMBER = "7a1d0c3e-0000-4000-8000-0000000002b1";
const BRUNO_MEMBER = "7a1d0c3e-0000-4000-8000-0000000002b2";
const CARLA_MEMBER = "7a1d0c3e-0000-4000-8000-0000000002b3";
const CARLA_HOOLI_MEMBER = "7a1d0c3e-0000-4000-8000-0000000002b4";

function actor(userId: string, memberId: string, role: Actor["role"]): Actor {
  return {
    userId: userIdSchema.parse(userId),
    organizationId: organizationIdSchema.parse(INITECH),
    memberId: memberId as Actor["memberId"],
    role,
  };
}

const anaAsOwner = actor(ANA, ANA_MEMBER, "owner");

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "deleteOrganization",
  () => {
    let database: Pool;

    beforeAll(() => {
      database = fixtureDatabase();
    });

    async function removeFixtures() {
      await database.query(`DELETE FROM "user" WHERE id = ANY($1)`, [
        [ANA, BRUNO, CARLA],
      ]);
      await database.query(`DELETE FROM organization WHERE id = ANY($1)`, [
        [INITECH, HOOLI],
      ]);
    }

    beforeEach(async () => {
      await removeFixtures();
      await database.query(
        `INSERT INTO "user" (id, name, email, "emailVerified")
         VALUES ($1, 'Ana', 'ana@deletion.test', true),
                ($2, 'Bruno', 'bruno@deletion.test', true),
                ($3, 'Carla', 'carla@deletion.test', true)`,
        [ANA, BRUNO, CARLA]
      );
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Initech', 'deletion-initech', now()),
                ($2, 'Hooli', 'deletion-hooli', now())`,
        [INITECH, HOOLI]
      );
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
         VALUES ($1, $5, $6, 'owner', now()),
                ($2, $5, $7, 'admin', now()),
                ($3, $5, $8, 'viewer', now()),
                ($4, $9, $8, 'viewer', now())`,
        [
          ANA_MEMBER,
          BRUNO_MEMBER,
          CARLA_MEMBER,
          CARLA_HOOLI_MEMBER,
          INITECH,
          ANA,
          BRUNO,
          CARLA,
          HOOLI,
        ]
      );
      await database.query(
        `INSERT INTO invitation ("organizationId", email, role, status, "expiresAt", "inviterId")
         VALUES ($1, 'dora@deletion.test', 'viewer', 'pending', now() + interval '7 days', $3),
                ($2, 'dora@deletion.test', 'admin', 'pending', now() + interval '7 days', $4)`,
        [INITECH, HOOLI, ANA, CARLA]
      );
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    async function remaining() {
      const organizations = await database.query<{ name: string }>(
        `SELECT name FROM organization WHERE id = ANY($1) ORDER BY name`,
        [[INITECH, HOOLI]]
      );
      const members = await database.query<{ id: string }>(
        `SELECT id FROM member WHERE "organizationId" = ANY($1) ORDER BY id`,
        [[INITECH, HOOLI]]
      );
      const invitations = await database.query<{ role: string }>(
        `SELECT role FROM invitation WHERE "organizationId" = ANY($1) ORDER BY role`,
        [[INITECH, HOOLI]]
      );
      const users = await database.query<{ name: string }>(
        `SELECT name FROM "user" WHERE id = ANY($1) ORDER BY name`,
        [[ANA, BRUNO, CARLA]]
      );
      return {
        organizations: organizations.rows.map(({ name }) => name),
        members: members.rows.map(({ id }) => id),
        invitations: invitations.rows.map(({ role }) => role),
        users: users.rows.map(({ name }) => name),
      };
    }

    const untouched = {
      organizations: ["Hooli", "Initech"],
      members: [ANA_MEMBER, BRUNO_MEMBER, CARLA_MEMBER, CARLA_HOOLI_MEMBER],
      invitations: ["admin", "viewer"],
      users: ["Ana", "Bruno", "Carla"],
    };

    it("deletes the Organization with its Members and Invitations, and nothing else", async () => {
      expect(await deleteOrganization(anaAsOwner, "Initech")).toEqual({
        kind: "ok",
      });

      expect(await remaining()).toEqual({
        organizations: ["Hooli"],
        members: [CARLA_HOOLI_MEMBER],
        invitations: ["admin"],
        users: ["Ana", "Bruno", "Carla"],
      });
    });

    it("refuses a confirmation that is not exactly the name", async () => {
      for (const confirmation of ["initech", "Initech ", "Hooli", ""]) {
        expect(await deleteOrganization(anaAsOwner, confirmation)).toEqual({
          kind: "error",
          code: "nameMismatch",
        });
      }

      expect(await remaining()).toEqual(untouched);
    });

    it("refuses an Admin", async () => {
      expect(
        await deleteOrganization(actor(BRUNO, BRUNO_MEMBER, "admin"), "Initech")
      ).toEqual({ kind: "error", code: "forbidden" });

      expect(await remaining()).toEqual(untouched);
    });

    it("refuses an Actor who is no longer the Owner", async () => {
      expect(
        await deleteOrganization(actor(BRUNO, BRUNO_MEMBER, "owner"), "Initech")
      ).toEqual({ kind: "error", code: "forbidden" });

      expect(await remaining()).toEqual(untouched);
    });
  }
);
