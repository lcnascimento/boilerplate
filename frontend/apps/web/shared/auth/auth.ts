import { betterAuth } from "better-auth";
import { APIError } from "better-auth/api";
import { nextCookies } from "better-auth/next-js";
import { jwt, organization } from "better-auth/plugins";
import { Pool } from "pg";

import { listMembersOldestFirst } from "./actor";
import { authRules, authRulesHook } from "./auth-rules";
import { parseIamEnv, type IamEnv } from "./env";
import { loginFailureHook } from "./login-failures";
import {
  INVITATION_EXPIRES_IN_DAYS,
  ORGANIZATION_LIMIT,
  PASSWORD_LENGTH,
} from "./policy";
import { roleHooks } from "./role-hooks";
import { ac, roles } from "./roles.gen";

const DAY_SECONDS = 60 * 60 * 24;

export const ACTOR_TOKEN_AUDIENCE = "boilerplate-api";

function createIam(env: IamEnv) {
  const database = new Pool({ connectionString: env.IAM_DATABASE_URL });
  const auth = betterAuth({
    baseURL: env.BETTER_AUTH_URL,
    secret: env.BETTER_AUTH_SECRET,
    database,
    emailAndPassword: {
      enabled: true,
      minPasswordLength: PASSWORD_LENGTH.min,
      maxPasswordLength: PASSWORD_LENGTH.max,
    },
    session: { expiresIn: 7 * DAY_SECONDS, updateAge: DAY_SECONDS },
    rateLimit: { storage: "database" },
    disabledPaths: ["/token"],
    hooks: { before: authRulesHook(authRules), after: loginFailureHook },
    advanced: {
      defaultCookieAttributes: {
        httpOnly: true,
        secure: true,
        sameSite: "strict",
      },
      database: { generateId: "uuid" },
    },
    databaseHooks: {
      session: {
        create: {
          before: async (session) => {
            const [oldest] = await listMembersOldestFirst(
              database,
              session.userId
            );
            if (oldest === undefined) return;

            return {
              data: {
                ...session,
                activeOrganizationId: oldest.organization.id,
              },
            };
          },
        },
      },
    },
    plugins: [
      jwt({
        disableSettingJwtHeader: true,
        jwks: {
          keyPairConfig: { alg: "EdDSA", crv: "Ed25519" },
          rotationInterval: 30 * DAY_SECONDS,
          gracePeriod: 7 * DAY_SECONDS,
        },
        jwt: {
          issuer: env.BETTER_AUTH_URL,
          audience: ACTOR_TOKEN_AUDIENCE,
        },
      }),
      organization({
        ac,
        roles,
        allowUserToCreateOrganization: true,
        organizationLimit: ORGANIZATION_LIMIT,
        disableOrganizationDeletion: true,
        invitationExpiresIn: INVITATION_EXPIRES_IN_DAYS * DAY_SECONDS,
        organizationHooks: {
          ...roleHooks(async (organizationId) => {
            const { rowCount } = await database.query(
              'SELECT 1 FROM member WHERE "organizationId" = $1 LIMIT 1',
              [organizationId]
            );
            return (rowCount ?? 0) > 0;
          }),
          beforeAcceptInvitation: async ({ user }) => {
            const memberships = await listMembersOldestFirst(database, user.id);
            if (memberships.length < ORGANIZATION_LIMIT) return;
            throw new APIError("FORBIDDEN", {
              code: "YOU_HAVE_REACHED_THE_MAXIMUM_NUMBER_OF_ORGANIZATIONS",
              message: "You have reached the maximum number of organizations",
            });
          },
        },
      }),
      nextCookies(),
    ],
  });

  return { auth, database };
}

type Iam = ReturnType<typeof createIam>;

let iam: Iam | undefined;

// Lazy: `next build` loads the routes without the runtime environment.
export function getIam(): Iam {
  iam ??= createIam(parseIamEnv(process.env));
  return iam;
}
