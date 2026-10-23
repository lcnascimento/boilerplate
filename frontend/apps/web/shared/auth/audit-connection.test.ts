import { sql } from "kysely";
import { Pool } from "pg";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { createUserOutsideSignUp } from "@/scripts/iam-seed/create-user";

import { organizationIdSchema, userIdSchema } from "./actor-ids";
import {
  createAuditedDatabase,
  currentConnectionSettings,
  hashIp,
  runAsActor,
} from "./audit-connection";
import { getIam } from "./auth";

const KEY = "a-test-ip-hash-key-that-is-long-enough";
const IP = "203.0.113.7";
const ADA = {
  userId: userIdSchema.parse("7a1d0c3e-0000-4000-8000-000000000301"),
  organizationId: organizationIdSchema.parse(
    "7a1d0c3e-0000-4000-8000-0000000003a1"
  ),
};
const fromIp = new Headers({ "x-forwarded-for": IP });

describe("hashIp", () => {
  it("is the HMAC-SHA256 of the IP under the key, in hex", () => {
    expect(hashIp(KEY, IP)).toBe(
      "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7"
    );
  });
});

describe("currentConnectionSettings", () => {
  it("is empty outside a request", () => {
    expect(currentConnectionSettings(KEY, {})).toEqual({
      actorId: "",
      actorOrganizationId: "",
      ipHash: "",
    });
  });

  it("carries the Actor and the hash of the forwarded IP inside runAsActor", async () => {
    await runAsActor(ADA, fromIp, () => {
      expect(currentConnectionSettings(KEY, {})).toEqual({
        actorId: "7a1d0c3e-0000-4000-8000-000000000301",
        actorOrganizationId: "7a1d0c3e-0000-4000-8000-0000000003a1",
        ipHash:
          "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
      });
      return Promise.resolve();
    });
  });
});

interface RecordedOrigin {
  actorId: string | null;
  actorOrganizationId: string | null;
  ipHash: string | null;
}

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "the audited connection",
  () => {
    const marker = `audit-${crypto.randomUUID().slice(0, 8)}`;
    const email = (name: string) => `${name}-${marker}@audit.test`;
    let database: Pool;

    beforeAll(() => {
      database = getIam().database;
    });

    afterAll(async () => {
      await database.query(`DELETE FROM "user" WHERE email LIKE $1`, [
        `%-${marker}@audit.test`,
      ]);
      await database.query(`DELETE FROM organization WHERE slug = $1`, [
        marker,
      ]);
      await database.end();
    });

    async function recordedOrigins(
      table: "user" | "account",
      operation: "INSERT" | "UPDATE",
      userId: string
    ): Promise<RecordedOrigin[]> {
      const eventType =
        table === "user"
          ? "boilerplate.iam.v1.UserCreated"
          : "boilerplate.iam.v1.PasswordChanged";
      const { rows } = await database.query<RecordedOrigin>(
        `SELECT actor_id::text AS "actorId",
                actor_organization_id::text AS "actorOrganizationId",
                ip_hash AS "ipHash"
         FROM identity_change
         WHERE table_name = $1 AND operation = $2 AND fields ->> $3 = $4
         UNION ALL
         SELECT substring(headers ->> 'ce_actorid' FROM 6),
                substring(headers ->> 'ce_actororganizationid' FROM 5),
                headers ->> 'ce_iphash'
         FROM watermill_iam_outbox,
              LATERAL (SELECT payload::jsonb -> 'metadata' AS headers) AS message
         WHERE headers ->> 'ce_type' = $5 AND headers ->> 'ce_subject' = 'user_' || $4`,
        [
          table,
          operation,
          table === "user" ? "id" : "userId",
          userId,
          eventType,
        ]
      );
      return rows;
    }

    it("records the Actor and the IP hash of an authenticated request", async () => {
      const { auth } = getIam();
      const password = "first-password-1";
      const { user } = await auth.api.signUpEmail({
        body: { name: "Ada", email: email("ada"), password },
        headers: fromIp,
      });
      const organization = await auth.api.createOrganization({
        body: { name: "Audit", slug: marker, userId: user.id },
      });
      const signedIn = await auth.api.signInEmail({
        body: { email: email("ada"), password },
        headers: fromIp,
        returnHeaders: true,
      });
      const cookie = signedIn.headers
        .getSetCookie()
        .map((setCookie) => setCookie.split(";")[0])
        .join("; ");

      await auth.api.changePassword({
        body: { currentPassword: password, newPassword: "second-password-2" },
        headers: new Headers({ cookie, "x-forwarded-for": IP }),
      });

      const ipHash = hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", IP);
      expect(await recordedOrigins("user", "INSERT", user.id)).toEqual([
        { actorId: null, actorOrganizationId: null, ipHash },
      ]);
      expect(await recordedOrigins("account", "UPDATE", user.id)).toEqual([
        { actorId: user.id, actorOrganizationId: organization.id, ipHash },
      ]);

      const { rows } = await database.query<{ leaks: number }>(
        `SELECT (SELECT count(*) FROM identity_change WHERE identity_change::text LIKE $1)
              + (SELECT count(*) FROM watermill_iam_outbox WHERE payload::text LIKE $1) AS leaks`,
        [`%${IP}%`]
      );
      expect(Number(rows[0]?.leaks)).toBe(0);
    });

    it("records no Actor nor IP hash outside a request", async () => {
      const context = await getIam().auth.$context;
      const userId = await createUserOutsideSignUp(context, {
        name: "Grace",
        email: email("grace"),
        password: "seed-password-1",
      });

      expect(await recordedOrigins("user", "INSERT", userId)).toEqual([
        { actorId: null, actorOrganizationId: null, ipHash: null },
      ]);
    });

    it("never hands a reused connection the Actor or the IP of an earlier request", async () => {
      const pool = new Pool({
        connectionString: process.env.IAM_DATABASE_URL,
        max: 1,
      });
      const db = createAuditedDatabase(pool, () =>
        currentConnectionSettings(KEY, {})
      );
      const read = async () => {
        const { rows } = await sql<{
          backend: number;
          actorId: string;
          actorOrganizationId: string;
          ipHash: string;
        }>`SELECT pg_backend_pid() AS backend,
                  current_setting('app.actor_id') AS "actorId",
                  current_setting('app.actor_organization_id') AS "actorOrganizationId",
                  current_setting('app.ip_hash') AS "ipHash"`.execute(db);
        const [row] = rows;
        if (row === undefined) throw new Error("no row");
        return row;
      };

      try {
        const inRequest = await runAsActor(ADA, fromIp, read);
        const afterRequest = await read();

        expect(afterRequest.backend).toBe(inRequest.backend);
        expect(inRequest).toMatchObject({
          actorId: "7a1d0c3e-0000-4000-8000-000000000301",
          ipHash:
            "61370b3ab3601445f17092312e647e82fb0adbd5c0dd53592b53f2f4fbd59cc7",
        });
        expect(afterRequest).toMatchObject({
          actorId: "",
          actorOrganizationId: "",
          ipHash: "",
        });
      } finally {
        await db.destroy();
      }
    });

    it("keeps Better Auth transactions atomic", async () => {
      const { adapter } = await getIam().auth.$context;
      const identifier = `${marker}-rolled-back`;

      await expect(
        adapter.transaction(async (trx) => {
          await trx.create({
            model: "verification",
            data: {
              identifier,
              value: "pending",
              expiresAt: new Date(Date.now() + 60_000),
            },
          });
          throw new Error("abort");
        })
      ).rejects.toThrow("abort");

      const { rowCount } = await database.query(
        `SELECT 1 FROM verification WHERE identifier = $1`,
        [identifier]
      );
      expect(rowCount).toBe(0);
    });
  }
);
