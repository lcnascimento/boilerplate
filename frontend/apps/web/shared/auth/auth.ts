import { betterAuth } from "better-auth";
import { nextCookies } from "better-auth/next-js";
import { jwt, organization } from "better-auth/plugins";
import { Pool } from "pg";

import { logger } from "@/shared/logging";

import { listMembersOldestFirst } from "./actor";
import { parseIamEnv, type IamEnv } from "./env";
import { loginFailureHook } from "./login-failures";
import { PASSWORD_LENGTH } from "./policy";
import { roleHooks } from "./role-hooks";
import { ac, roles } from "./roles.gen";
import { signUpOrganizationName } from "./sign-up-body";
import { ensureSignUpOrganization as ensureOrganization } from "./sign-up-organization";

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
    user: {
      additionalFields: {
        pendingOrganizationName: {
          type: "string",
          required: false,
          input: false,
          returned: false,
        },
      },
    },
    session: { expiresIn: 7 * DAY_SECONDS, updateAge: DAY_SECONDS },
    rateLimit: { storage: "database" },
    disabledPaths: ["/token"],
    hooks: { after: loginFailureHook },
    advanced: {
      defaultCookieAttributes: {
        httpOnly: true,
        secure: true,
        sameSite: "strict",
      },
      database: { generateId: "uuid" },
    },
    databaseHooks: {
      user: {
        create: {
          before: (user, context) =>
            Promise.resolve({
              data: {
                ...user,
                pendingOrganizationName: signUpOrganizationName(context),
              },
            }),
          after: async (user) => {
            try {
              await ensureSignUpOrganization(user.id);
            } catch (error) {
              logger.error(
                { err: error, userId: user.id },
                "creating the sign-up organization failed"
              );
            }
          },
        },
      },
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
        allowUserToCreateOrganization: false,
        organizationHooks: roleHooks(async (organizationId) => {
          const { rowCount } = await database.query(
            'SELECT 1 FROM member WHERE "organizationId" = $1 LIMIT 1',
            [organizationId]
          );
          return (rowCount ?? 0) > 0;
        }),
      }),
      nextCookies(),
    ],
  });

  function ensureSignUpOrganization(userId: string): Promise<void> {
    return ensureOrganization(
      database,
      (body) => auth.api.createOrganization({ body }),
      userId
    );
  }

  return { auth, database, ensureSignUpOrganization };
}

type Iam = ReturnType<typeof createIam>;

let iam: Iam | undefined;

// Lazy: `next build` loads the routes without the runtime environment.
export function getIam(): Iam {
  iam ??= createIam(parseIamEnv(process.env));
  return iam;
}
