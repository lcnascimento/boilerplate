import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { userIdSchema } from "./actor-ids";
import { hashIp } from "./audit-connection";
import { getIam } from "./auth";
import { createOrganizationFor } from "./organization-creation";

import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const IP = "203.0.113.9";

interface RecordedChange {
  type: string;
  correlationId: string;
  actorId: string | null;
  actorOrganizationId: string | null;
  ipHash: string | null;
}

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "createOrganizationFor",
  () => {
    const marker = `creation-${crypto.randomUUID().slice(0, 8)}`;
    const email = `ada-${marker}@creation.test`;
    let database: Pool;

    beforeAll(() => {
      database = getIam().database;
    });

    afterAll(async () => {
      await database.query(`DELETE FROM "user" WHERE email = $1`, [email]);
      await database.query(`DELETE FROM organization WHERE slug = $1`, [
        marker,
      ]);
      await database.end();
    });

    async function recordedChanges(
      organizationId: string
    ): Promise<RecordedChange[]> {
      const { rows } = await database.query<RecordedChange>(
        `SELECT CASE table_name WHEN 'organization' THEN 'OrganizationCreated' ELSE 'MemberAdded' END AS type,
                transaction_id::text AS "correlationId",
                actor_id::text AS "actorId",
                actor_organization_id::text AS "actorOrganizationId",
                ip_hash AS "ipHash"
         FROM identity_change
         WHERE operation = 'INSERT'
           AND ((table_name = 'organization' AND fields ->> 'id' = $1)
             OR (table_name = 'member' AND fields ->> 'organizationId' = $1))
         UNION ALL
         SELECT substring(headers ->> 'ce_type' FROM '[^.]+$'),
                headers ->> 'ce_correlationid',
                substring(headers ->> 'ce_actorid' FROM 6),
                substring(headers ->> 'ce_actororganizationid' FROM 5),
                headers ->> 'ce_iphash'
         FROM watermill_iam_outbox,
              LATERAL (SELECT payload::jsonb -> 'metadata' AS headers) AS message
         WHERE headers ->> 'ce_type' IN ('boilerplate.iam.v1.OrganizationCreated', 'boilerplate.iam.v1.MemberAdded')
           AND headers ->> 'ce_subject' = 'org_' || $1
         ORDER BY type DESC`,
        [organizationId]
      );
      return rows;
    }

    it("creates the Organization and its Owner in one transaction, as the creator", async () => {
      const { auth } = getIam();
      const password = "creation-password-1";
      await auth.api.signUpEmail({ body: { name: "Ada", email, password } });
      const signedIn = await auth.api.signInEmail({
        body: { email, password },
        returnHeaders: true,
      });
      const headers = new Headers({
        cookie: signedIn.headers
          .getSetCookie()
          .map((setCookie) => setCookie.split(";")[0])
          .join("; "),
        "x-forwarded-for": IP,
      });
      const userId = userIdSchema.parse(signedIn.response.user.id);

      const organization = await createOrganizationFor(
        { userId, organizationId: null },
        headers,
        { name: "Creation", slug: marker }
      );

      const changes = await recordedChanges(organization.id);
      const ipHash = hashIp(process.env.AUDIT_IP_HASH_KEY ?? "", IP);
      expect(changes).toEqual([
        {
          type: "OrganizationCreated",
          correlationId: changes[0]?.correlationId,
          actorId: userId,
          actorOrganizationId: null,
          ipHash,
        },
        {
          type: "MemberAdded",
          correlationId: changes[0]?.correlationId,
          actorId: userId,
          actorOrganizationId: null,
          ipHash,
        },
      ]);
    });
  }
);
