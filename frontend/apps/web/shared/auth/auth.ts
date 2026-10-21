import { betterAuth } from "better-auth";
import { APIError } from "better-auth/api";
import { nextCookies } from "better-auth/next-js";
import { jwt, organization } from "better-auth/plugins";
import { Pool } from "pg";
import { z } from "zod";

import { logger } from "@/shared/logging";

import { parseIamEnv, type IamEnv } from "./env";
import { ORGANIZATION_NAME_MAX_LENGTH, PASSWORD_LENGTH } from "./policy";
import { ac, roles } from "./roles.gen";
import { ensureSignUpOrganization as ensureOrganization } from "./sign-up-organization";

const DAY_SECONDS = 60 * 60 * 24;

export const ACTOR_TOKEN_AUDIENCE = "boilerplate-api";

const signUpBodySchema = z.object({
  organizationName: z
    .string()
    .trim()
    .min(1)
    .max(ORGANIZATION_NAME_MAX_LENGTH)
    .optional(),
});

// Absent for sign-ups that do not come from the form, such as the seed, which places its Users itself.
function parseOrganizationName(body: unknown): string | null {
  const parsed = signUpBodySchema.safeParse(body ?? {});
  if (!parsed.success) {
    throw new APIError("BAD_REQUEST", {
      code: "INVALID_ORGANIZATION_NAME",
      message: "Invalid organization name",
    });
  }
  return parsed.data.organizationName ?? null;
}

function rejectUnknownRoles(role: string): Promise<void> {
  const unknown = role.split(",").filter((name) => !Object.hasOwn(roles, name));
  if (unknown.length === 0) return Promise.resolve();
  return Promise.reject(
    new APIError("BAD_REQUEST", {
      code: "UNKNOWN_ROLE",
      message: `Unknown role: ${unknown.join(", ")}`,
    })
  );
}

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
                pendingOrganizationName: parseOrganizationName(context?.body),
              },
            }),
          after: async (user) => {
            try {
              await ensureSignUpOrganization(user.id);
            } catch (error) {
              // The (dashboard) layout retries, so the sign-up itself still succeeds.
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
          before: async (session, context) => {
            const adapter = context?.context.adapter;
            if (adapter === undefined) return;

            const [oldest] = await adapter.findMany<{ organizationId: string }>(
              {
                model: "member",
                where: [{ field: "userId", value: session.userId }],
                sortBy: { field: "createdAt", direction: "asc" },
                limit: 1,
              }
            );
            if (oldest === undefined) return;

            return {
              data: { ...session, activeOrganizationId: oldest.organizationId },
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
        organizationHooks: {
          beforeAddMember: ({ member }) => rejectUnknownRoles(member.role),
          beforeUpdateMemberRole: ({ newRole }) => rejectUnknownRoles(newRole),
          beforeCreateInvitation: ({ invitation }) =>
            rejectUnknownRoles(invitation.role),
        },
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

// Lazy, so that `next build` can load the routes without the runtime environment.
export function getIam(): Iam {
  iam ??= createIam(parseIamEnv(process.env));
  return iam;
}
