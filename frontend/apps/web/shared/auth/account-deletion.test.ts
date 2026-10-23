import {
  afterAll,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { deleteAccount, accountDeletionBlockers } from "./account-deletion";
import { organizationIdSchema, userIdSchema } from "./actor-ids";
import { hashIp } from "./audit-connection";
import { getIam } from "./auth";

import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const ANA = "7a1d0c3e-0000-4000-8000-000000000701";
const BRUNO = "7a1d0c3e-0000-4000-8000-000000000702";
const INITECH = "7a1d0c3e-0000-4000-8000-0000000007a1";
const ANA_SOLO = "7a1d0c3e-0000-4000-8000-0000000007a2";
const BRUNO_SOLO = "7a1d0c3e-0000-4000-8000-0000000007a3";
const ANA_INITECH = "7a1d0c3e-0000-4000-8000-0000000007b1";
const BRUNO_INITECH = "7a1d0c3e-0000-4000-8000-0000000007b2";
const ANA_SOLO_OWNER = "7a1d0c3e-0000-4000-8000-0000000007b3";
const BRUNO_SOLO_OWNER = "7a1d0c3e-0000-4000-8000-0000000007b4";
const USERS = [ANA, BRUNO];
const ORGANIZATIONS = [INITECH, ANA_SOLO, BRUNO_SOLO];
const PASSWORD = "account-deletion-password";

const fromIp = new Headers({ "x-forwarded-for": "198.51.100.1" });

const ana = {
  userId: userIdSchema.parse(ANA),
  organizationId: organizationIdSchema.parse(INITECH),
};
const bruno = {
  userId: userIdSchema.parse(BRUNO),
  organizationId: organizationIdSchema.parse(BRUNO_SOLO),
};

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "account deletion",
  () => {
    let database: Pool;

    beforeAll(() => {
      database = getIam().database;
    });

    async function removeFixtures() {
      await database.query(`DELETE FROM "user" WHERE id = ANY($1)`, [USERS]);
      await database.query(`DELETE FROM organization WHERE id = ANY($1)`, [
        ORGANIZATIONS,
      ]);
    }

    beforeEach(async () => {
      await removeFixtures();
      const { password } = await getIam().auth.$context;
      const hash = await password.hash(PASSWORD);
      await database.query(
        `INSERT INTO "user" (id, name, email, "emailVerified")
         VALUES ($1, 'Ana', 'ana@account-deletion.test', true),
                ($2, 'Bruno', 'bruno@account-deletion.test', true)`,
        USERS
      );
      await database.query(
        `INSERT INTO account (id, "accountId", "providerId", "userId", password, "createdAt", "updatedAt")
         VALUES (gen_random_uuid(), $1::text, 'credential', $1::uuid, $3, now(), now()),
                (gen_random_uuid(), $2::text, 'credential', $2::uuid, $3, now(), now())`,
        [...USERS, hash]
      );
      await database.query(
        `INSERT INTO session (id, token, "userId", "expiresAt", "createdAt", "updatedAt")
         VALUES (gen_random_uuid(), 'account-deletion-ana', $1, now() + interval '1 day', now(), now()),
                (gen_random_uuid(), 'account-deletion-bruno', $2, now() + interval '1 day', now(), now())`,
        USERS
      );
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Initech', 'account-deletion-initech', now()),
                ($2, 'Ana Solo', 'account-deletion-ana-solo', now()),
                ($3, 'Bruno Solo', 'account-deletion-bruno-solo', now())`,
        ORGANIZATIONS
      );
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
         VALUES ($1, $5, $8, 'owner', now()),
                ($2, $5, $9, 'admin', now()),
                ($3, $6, $8, 'owner', now()),
                ($4, $7, $9, 'owner', now())`,
        [
          ANA_INITECH,
          BRUNO_INITECH,
          ANA_SOLO_OWNER,
          BRUNO_SOLO_OWNER,
          INITECH,
          ANA_SOLO,
          BRUNO_SOLO,
          ANA,
          BRUNO,
        ]
      );
      await database.query(
        `INSERT INTO invitation ("organizationId", email, role, status, "expiresAt", "inviterId")
         VALUES ($1, 'dora@account-deletion.test', 'viewer', 'pending', now() + interval '7 days', $2),
                ($1, 'enzo@account-deletion.test', 'admin', 'pending', now() + interval '7 days', $3)`,
        [INITECH, ANA, BRUNO]
      );
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    async function remaining() {
      const query = async (sql: string, values: unknown[]) =>
        (await database.query<{ value: string }>(sql, values)).rows.map(
          ({ value }) => value
        );
      return {
        users: await query(
          `SELECT name AS value FROM "user" WHERE id = ANY($1) ORDER BY name`,
          [USERS]
        ),
        accounts: await query(
          `SELECT "userId" AS value FROM account WHERE "userId" = ANY($1) ORDER BY "userId"`,
          [USERS]
        ),
        sessions: await query(
          `SELECT token AS value FROM session WHERE "userId" = ANY($1) ORDER BY token`,
          [USERS]
        ),
        organizations: await query(
          `SELECT name AS value FROM organization WHERE id = ANY($1) ORDER BY name`,
          [ORGANIZATIONS]
        ),
        members: await query(
          `SELECT id AS value FROM member WHERE "organizationId" = ANY($1) ORDER BY id`,
          [ORGANIZATIONS]
        ),
        invitations: await query(
          `SELECT email AS value FROM invitation WHERE "organizationId" = ANY($1) ORDER BY email`,
          [ORGANIZATIONS]
        ),
      };
    }

    const untouched = {
      users: ["Ana", "Bruno"],
      accounts: [ANA, BRUNO],
      sessions: ["account-deletion-ana", "account-deletion-bruno"],
      organizations: ["Ana Solo", "Bruno Solo", "Initech"],
      members: [ANA_INITECH, BRUNO_INITECH, ANA_SOLO_OWNER, BRUNO_SOLO_OWNER],
      invitations: ["dora@account-deletion.test", "enzo@account-deletion.test"],
    };

    it("lists the Organizations an Owner shares with other Members", async () => {
      expect(await accountDeletionBlockers(ana.userId)).toEqual([
        { id: INITECH, name: "Initech" },
      ]);
      expect(await accountDeletionBlockers(bruno.userId)).toEqual([]);
    });

    it("refuses the Owner of an Organization with other Members, and deletes nothing", async () => {
      expect(await deleteAccount(ana, fromIp, PASSWORD)).toEqual({
        kind: "error",
        code: "ownsSharedOrganizations",
        organizations: [{ id: INITECH, name: "Initech" }],
      });

      expect(await remaining()).toEqual(untouched);
    });

    it("refuses a wrong password, and deletes nothing", async () => {
      expect(await deleteAccount(bruno, fromIp, "not-the-password")).toEqual({
        kind: "error",
        code: "invalidPassword",
      });

      expect(await remaining()).toEqual(untouched);
    });

    it("deletes the User with the sole-Member Organizations, and keeps the shared ones", async () => {
      expect(await deleteAccount(bruno, fromIp, PASSWORD)).toEqual({
        kind: "ok",
      });

      expect(await remaining()).toEqual({
        users: ["Ana"],
        accounts: [ANA],
        sessions: ["account-deletion-ana"],
        organizations: ["Ana Solo", "Initech"],
        members: [ANA_INITECH, ANA_SOLO_OWNER],
        invitations: ["dora@account-deletion.test"],
      });
    });

    it("records the deleting User, their active Organization and the IP hash on every change", async () => {
      const ip = `10.${crypto.getRandomValues(new Uint8Array(3)).join(".")}`;
      const ipHash = hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", ip);

      await deleteAccount(
        bruno,
        new Headers({ "x-forwarded-for": ip }),
        PASSWORD
      );

      const { rows } = await database.query(
        `SELECT CASE table_name || ' ' || operation
                  WHEN 'user DELETE' THEN 'UserDeleted'
                  WHEN 'session DELETE' THEN 'SessionEnded'
                  WHEN 'organization DELETE' THEN 'OrganizationDeleted'
                  WHEN 'member DELETE' THEN 'MemberRemoved'
                  ELSE table_name || ' ' || operation
                END AS type,
                actor_id::text AS "actorId",
                actor_organization_id::text AS "actorOrganizationId"
         FROM identity_change
         WHERE ip_hash = $1
         UNION ALL
         SELECT substring(headers ->> 'ce_type' FROM 20),
                substring(headers ->> 'ce_actorid' FROM 6),
                substring(headers ->> 'ce_actororganizationid' FROM 5)
         FROM watermill_iam_outbox,
              LATERAL (SELECT payload::jsonb -> 'metadata' AS headers) AS message
         WHERE headers ->> 'ce_iphash' = $1
         ORDER BY type DESC`,
        [ipHash]
      );

      const asBruno = { actorId: BRUNO, actorOrganizationId: BRUNO_SOLO };
      expect(rows).toEqual([
        { type: "UserDeleted", ...asBruno },
        { type: "SessionEnded", ...asBruno },
        { type: "OrganizationDeleted", ...asBruno },
        { type: "MemberRemoved", ...asBruno },
        { type: "MemberRemoved", ...asBruno },
      ]);
    });
  }
);
