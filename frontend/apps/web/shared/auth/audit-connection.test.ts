import { sql } from "kysely";
import { Pool } from "pg";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { createUserOutsideSignUp } from "@/scripts/iam-seed/create-user";

import { organizationIdSchema, userIdSchema } from "./actor-ids";
import {
  createAuditedDatabase,
  currentConnectionSettings,
  hashIp,
  runAsActor,
} from "./audit-connection";
import { getIam } from "./auth";
import { acceptInvitationFor } from "./invitation-acceptance";
import { createOrganizationFor } from "./organization-creation";
import { fixtureDatabase } from "./test-database";

vi.mock("server-only", () => ({}));

const KEY = "a-test-ip-hash-key-that-is-long-enough";
const IP = "203.0.113.7";
const ADA = {
  userId: userIdSchema.parse("7a1d0c3e-0000-4000-8000-000000000301"),
  organizationId: organizationIdSchema.parse(
    "7a1d0c3e-0000-4000-8000-0000000003a1"
  ),
};
const fromIp = new Headers({ "x-forwarded-for": IP });

function randomOctet(): number {
  return crypto.getRandomValues(new Uint8Array(1))[0] ?? 0;
}

function randomIp(): string {
  return `10.${String(randomOctet())}.${String(randomOctet())}.${String(randomOctet())}`;
}

function cookieOf(headers: Headers): string {
  return headers
    .getSetCookie()
    .map((setCookie) => setCookie.split(";")[0])
    .join("; ");
}

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
      database = fixtureDatabase();
    });

    afterAll(async () => {
      await database.query(`DELETE FROM "user" WHERE email LIKE $1`, [
        `%-${marker}@audit.test`,
      ]);
      await database.query(`DELETE FROM organization WHERE slug LIKE $1`, [
        `${marker}%`,
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

    async function recordedEventsOf(ipHash: string): Promise<string[]> {
      const { rows } = await database.query<{ type: string }>(
        `SELECT CASE table_name || ' ' || operation
                  WHEN 'session INSERT' THEN 'SessionStarted'
                  WHEN 'session DELETE' THEN 'SessionEnded'
                  WHEN 'sign_in FAILED' THEN 'SignInFailed'
                END AS type
         FROM identity_change
         WHERE ip_hash = $1 AND table_name IN ('session', 'sign_in')
         UNION ALL
         SELECT substring(headers ->> 'ce_type' FROM '[^.]+$')
         FROM watermill_iam_outbox,
              LATERAL (SELECT payload::jsonb -> 'metadata' AS headers) AS message
         WHERE headers ->> 'ce_iphash' = $1
         ORDER BY type`,
        [ipHash]
      );
      return rows.map(({ type }) => type);
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
      const cookie = cookieOf(signedIn.headers);

      await auth.api.changePassword({
        body: { currentPassword: password, newPassword: "second-password-2" },
        headers: new Headers({ cookie, "x-forwarded-for": IP }),
      });

      const ipHash = hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", IP);
      expect([
        [{ actorId: null, actorOrganizationId: null, ipHash }],
        [{ actorId: user.id, actorOrganizationId: null, ipHash }],
      ]).toContainEqual(await recordedOrigins("user", "INSERT", user.id));
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

    it("records the sessions and the failed sign-ins of a request with its IP hash, never the email", async () => {
      const { auth } = getIam();
      const password = "hedy-password-1";
      const ip = `10.${String(randomOctet())}.${String(randomOctet())}.${String(randomOctet())}`;
      const fromHedysIp = new Headers({ "x-forwarded-for": ip });
      await auth.api.signUpEmail({
        body: { name: "Hedy", email: email("hedy"), password },
      });

      const signedIn = await auth.api.signInEmail({
        body: { email: email("hedy"), password },
        headers: fromHedysIp,
        returnHeaders: true,
      });
      await auth.api.signOut({
        headers: new Headers({
          cookie: cookieOf(signedIn.headers),
          "x-forwarded-for": ip,
        }),
      });
      await expect(
        auth.api.signInEmail({
          body: { email: email("hedy"), password: "wrong-password" },
          headers: fromHedysIp,
        })
      ).rejects.toThrow();
      await expect(
        auth.api.signInEmail({
          body: { email: email("nobody"), password },
          headers: fromHedysIp,
        })
      ).rejects.toThrow();

      expect(
        await recordedEventsOf(hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", ip))
      ).toEqual([
        "SessionEnded",
        "SessionStarted",
        "SignInFailed",
        "SignInFailed",
      ]);

      const { rows } = await database.query<{ leaks: number }>(
        `SELECT (SELECT count(*) FROM identity_change WHERE identity_change::text LIKE $1 OR identity_change::text LIKE $2)
              + (SELECT count(*) FROM watermill_iam_outbox WHERE payload::text LIKE $1 OR payload::text LIKE $2) AS leaks`,
        [`%${marker}%`, `%${ip}%`]
      );
      expect(Number(rows[0]?.leaks)).toBe(0);
    });

    async function recordedOrganizationEventsOf(ip: string) {
      const { rows } = await database.query<{
        type: string;
        actorId: string | null;
        actorOrganizationId: string | null;
      }>(
        `SELECT CASE table_name || ' ' || operation
                  WHEN 'organization INSERT' THEN 'OrganizationCreated'
                  WHEN 'organization UPDATE' THEN 'OrganizationRenamed'
                  WHEN 'member INSERT' THEN 'MemberAdded'
                  WHEN 'member UPDATE' THEN 'MemberRoleChanged'
                  WHEN 'member DELETE' THEN 'MemberRemoved'
                  WHEN 'invitation INSERT' THEN 'InvitationCreated'
                  WHEN 'invitation UPDATE' THEN
                    CASE fields ->> 'status'
                      WHEN 'accepted' THEN 'InvitationAccepted'
                      WHEN 'rejected' THEN 'InvitationDeclined'
                      WHEN 'canceled' THEN 'InvitationCanceled'
                    END
                END AS type,
                actor_id::text AS "actorId",
                actor_organization_id::text AS "actorOrganizationId"
         FROM identity_change
         WHERE ip_hash = $1 AND table_name IN ('organization', 'member', 'invitation')
         UNION ALL
         SELECT type,
                NULLIF(substring(headers ->> 'ce_actorid' FROM 6), ''),
                NULLIF(substring(headers ->> 'ce_actororganizationid' FROM 5), '')
         FROM watermill_iam_outbox,
              LATERAL (SELECT payload::jsonb -> 'metadata' AS headers) AS message,
              LATERAL (SELECT substring(headers ->> 'ce_type' FROM '[^.]+$') AS type) AS short
         WHERE headers ->> 'ce_iphash' = $1 AND type ~ '^(Organization|Member|Invitation)'
         ORDER BY type, "actorId"`,
        [hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", ip)]
      );
      return rows;
    }

    it("records the Actor, their Organization and the IP hash of Organization, Member and Invitation changes", async () => {
      const { auth } = getIam();
      const password = "lifecycle-password-1";
      const ips = { olga: randomIp(), vera: randomIp(), wes: randomIp() };
      async function signedUp(name: "olga" | "vera" | "wes") {
        const { user } = await auth.api.signUpEmail({
          body: { name, email: email(name), password },
        });
        const signedIn = await auth.api.signInEmail({
          body: { email: email(name), password },
          returnHeaders: true,
        });
        return {
          id: userIdSchema.parse(user.id),
          headers: new Headers({
            cookie: cookieOf(signedIn.headers),
            "x-forwarded-for": ips[name],
          }),
        };
      }
      const olga = await signedUp("olga");
      const vera = await signedUp("vera");
      const wes = await signedUp("wes");

      const organization = await createOrganizationFor(
        { userId: olga.id, organizationId: null },
        olga.headers,
        { name: "Lifecycle", slug: `${marker}-lifecycle` }
      );
      const organizationId = organization.id;
      const invite = (invited: string) =>
        auth.api.createInvitation({
          headers: olga.headers,
          body: { email: invited, role: "viewer", organizationId },
        });
      const toVera = await invite(email("vera"));
      const toWes = await invite(email("wes"));
      const toZe = await invite(email("ze"));
      await acceptInvitationFor(
        { userId: vera.id, organizationId: null },
        vera.headers,
        toVera.id
      );
      await auth.api.rejectInvitation({
        headers: wes.headers,
        body: { invitationId: toWes.id },
      });
      await auth.api.cancelInvitation({
        headers: olga.headers,
        body: { invitationId: toZe.id },
      });
      await auth.api.updateOrganization({
        headers: olga.headers,
        body: { organizationId, data: { name: "Lifecycle renamed" } },
      });
      const { members } = await auth.api.listMembers({
        headers: olga.headers,
        query: { organizationId },
      });
      const veraMemberId = members.find(({ userId }) => userId === vera.id)?.id;
      if (veraMemberId === undefined) throw new Error("Vera is no Member");
      await auth.api.updateMemberRole({
        headers: olga.headers,
        body: { memberId: veraMemberId, organizationId, role: "admin" },
      });
      await auth.api.removeMember({
        headers: olga.headers,
        body: { memberIdOrEmail: veraMemberId, organizationId },
      });

      const asOlga = { actorId: olga.id, actorOrganizationId: organizationId };
      const asOlgaCreating = { actorId: olga.id, actorOrganizationId: null };
      expect(await recordedOrganizationEventsOf(ips.olga)).toEqual([
        { type: "InvitationCanceled", ...asOlga },
        { type: "InvitationCreated", ...asOlga },
        { type: "InvitationCreated", ...asOlga },
        { type: "InvitationCreated", ...asOlga },
        { type: "MemberAdded", ...asOlgaCreating },
        { type: "MemberRemoved", ...asOlga },
        { type: "MemberRoleChanged", ...asOlga },
        { type: "OrganizationCreated", ...asOlgaCreating },
        { type: "OrganizationRenamed", ...asOlga },
      ]);
      const asVera = { actorId: vera.id, actorOrganizationId: null };
      expect(await recordedOrganizationEventsOf(ips.vera)).toEqual([
        { type: "InvitationAccepted", ...asVera },
        { type: "MemberAdded", ...asVera },
      ]);
      expect(await recordedOrganizationEventsOf(ips.wes)).toEqual([
        {
          type: "InvitationDeclined",
          actorId: wes.id,
          actorOrganizationId: null,
        },
      ]);
    });

    it("records no Actor nor IP hash outside a request", async () => {
      const context = await getIam().auth.$context;
      const userId = await createUserOutsideSignUp(context, {
        name: "Grace",
        email: email("grace"),
        password: "seed-password-1",
      });

      expect([
        [{ actorId: null, actorOrganizationId: null, ipHash: null }],
        [{ actorId: userId, actorOrganizationId: null, ipHash: null }],
      ]).toContainEqual(await recordedOrigins("user", "INSERT", userId));
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
