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
import { getIam } from "./auth";
import { transferOwnership } from "./ownership";

import type { Actor } from "./actor";
import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const ANA = "7a1d0c3e-0000-4000-8000-000000000101";
const BRUNO = "7a1d0c3e-0000-4000-8000-000000000102";
const CARLA = "7a1d0c3e-0000-4000-8000-000000000103";
const INITECH = "7a1d0c3e-0000-4000-8000-0000000001a1";
const HOOLI = "7a1d0c3e-0000-4000-8000-0000000001a2";
const ANA_MEMBER = "7a1d0c3e-0000-4000-8000-0000000001b1";
const BRUNO_MEMBER = "7a1d0c3e-0000-4000-8000-0000000001b2";
const CARLA_MEMBER = "7a1d0c3e-0000-4000-8000-0000000001b3";
const CARLA_HOOLI_MEMBER = "7a1d0c3e-0000-4000-8000-0000000001b4";

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
  "transferOwnership",
  () => {
    let database: Pool;

    beforeAll(() => {
      database = getIam().database;
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
         VALUES ($1, 'Ana', 'ana@ownership.test', true),
                ($2, 'Bruno', 'bruno@ownership.test', true),
                ($3, 'Carla', 'carla@ownership.test', true)`,
        [ANA, BRUNO, CARLA]
      );
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Initech', 'ownership-initech', now()),
                ($2, 'Hooli', 'ownership-hooli', now())`,
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
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    async function initechRoles() {
      const { rows } = await database.query<{ name: string; role: string }>(
        `SELECT u.name, m.role FROM member m JOIN "user" u ON u.id = m."userId"
          WHERE m."organizationId" = $1 ORDER BY u.name`,
        [INITECH]
      );
      return rows;
    }

    const untouched = [
      { name: "Ana", role: "owner" },
      { name: "Bruno", role: "admin" },
      { name: "Carla", role: "viewer" },
    ];

    it("hands the Organization to a Viewer and keeps the Owner as Admin", async () => {
      expect(await transferOwnership(anaAsOwner, CARLA_MEMBER)).toEqual({
        kind: "ok",
      });

      expect(await initechRoles()).toEqual([
        { name: "Ana", role: "admin" },
        { name: "Bruno", role: "admin" },
        { name: "Carla", role: "owner" },
      ]);
    });

    it("hands the Organization to an Admin", async () => {
      expect(await transferOwnership(anaAsOwner, BRUNO_MEMBER)).toEqual({
        kind: "ok",
      });

      expect(await initechRoles()).toEqual([
        { name: "Ana", role: "admin" },
        { name: "Bruno", role: "owner" },
        { name: "Carla", role: "viewer" },
      ]);
    });

    it("refuses an Admin", async () => {
      expect(
        await transferOwnership(
          actor(BRUNO, BRUNO_MEMBER, "admin"),
          CARLA_MEMBER
        )
      ).toEqual({ kind: "error", code: "forbidden" });

      expect(await initechRoles()).toEqual(untouched);
    });

    it("promotes nobody when the Actor is no longer the Owner", async () => {
      expect(
        await transferOwnership(
          actor(BRUNO, BRUNO_MEMBER, "owner"),
          CARLA_MEMBER
        )
      ).toEqual({ kind: "error", code: "forbidden" });

      expect(await initechRoles()).toEqual(untouched);
    });

    it("refuses a Member of another Organization", async () => {
      expect(await transferOwnership(anaAsOwner, CARLA_HOOLI_MEMBER)).toEqual({
        kind: "error",
        code: "invalidInput",
      });

      expect(await initechRoles()).toEqual(untouched);
    });

    it("refuses the Owner themselves", async () => {
      expect(await transferOwnership(anaAsOwner, ANA_MEMBER)).toEqual({
        kind: "error",
        code: "invalidInput",
      });

      expect(await initechRoles()).toEqual(untouched);
    });
  }
);
