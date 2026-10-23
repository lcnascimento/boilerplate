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
import { fixtureDatabase } from "./test-database";

import type { Pool } from "pg";

vi.mock("server-only", () => ({}));

const RITA = {
  name: "Rita",
  email: "rita@password-change.test",
  password: "rita-password",
};
const INITECH = "7a1d0c3e-0000-4000-8000-000001930001";
const HOOLI = "7a1d0c3e-0000-4000-8000-000001930002";

function cookieOf(headers: Headers): Headers {
  const cookie = headers
    .getSetCookie()
    .map((setCookie) => setCookie.split(";")[0])
    .join("; ");
  return new Headers({ cookie });
}

describe.skipIf(process.env.IAM_DATABASE_URL === undefined)(
  "changing the password",
  () => {
    let database: Pool;

    async function removeFixtures() {
      await database.query(`DELETE FROM "user" WHERE email = $1`, [RITA.email]);
      await database.query(`DELETE FROM organization WHERE id = ANY($1)`, [
        [INITECH, HOOLI],
      ]);
    }

    beforeAll(() => {
      database = fixtureDatabase();
    });

    beforeEach(async () => {
      await removeFixtures();
      const { user } = await getIam().auth.api.signUpEmail({ body: RITA });
      await database.query(
        `INSERT INTO organization (id, name, slug, "createdAt")
         VALUES ($1, 'Initech', 'password-change-initech', now()),
                ($2, 'Hooli', 'password-change-hooli', now())`,
        [INITECH, HOOLI]
      );
      await database.query(
        `INSERT INTO member (id, "organizationId", "userId", role, "createdAt")
         VALUES (gen_random_uuid(), $1, $3, 'owner', '2026-01-01T00:00:00Z'),
                (gen_random_uuid(), $2, $3, 'owner', '2026-02-01T00:00:00Z')`,
        [INITECH, HOOLI, user.id]
      );
    });

    afterAll(async () => {
      await removeFixtures();
      await database.end();
    });

    it("keeps the active Organization in the session that replaces the current one", async () => {
      const { auth } = getIam();
      const signedIn = await auth.api.signInEmail({
        body: { email: RITA.email, password: RITA.password },
        returnHeaders: true,
      });
      const headers = cookieOf(signedIn.headers);
      await auth.api.setActiveOrganization({
        headers,
        body: { organizationId: HOOLI },
      });

      const changed = await auth.api.changePassword({
        headers,
        body: {
          currentPassword: RITA.password,
          newPassword: "rita-new-password",
          revokeOtherSessions: true,
        },
        returnHeaders: true,
      });

      const session = await auth.api.getSession({
        headers: cookieOf(changed.headers),
      });
      expect(session?.session.activeOrganizationId).toBe(HOOLI);
    });

    it("starts a new sign-in in the oldest Organization", async () => {
      const { auth } = getIam();
      const signedIn = await auth.api.signInEmail({
        body: { email: RITA.email, password: RITA.password },
        returnHeaders: true,
      });

      const session = await auth.api.getSession({
        headers: cookieOf(signedIn.headers),
      });
      expect(session?.session.activeOrganizationId).toBe(INITECH);
    });
  }
);
