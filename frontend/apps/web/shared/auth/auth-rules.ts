import { createAuthMiddleware } from "better-auth/api";
import { z } from "zod";

export type AuthAction =
  | { kind: "signUp"; email: string }
  | { kind: "signIn"; email: string }
  | { kind: "createOrganization"; name: string }
  | { kind: "createInvitation"; email: string };

// A rule refuses an action by throwing a Better Auth APIError, which the caller receives as is.
export type AuthRule = (action: AuthAction) => Promise<void>;

export const authRules: readonly AuthRule[] = [];

const emailBodySchema = z.object({ email: z.string() });
const organizationBodySchema = z.object({ name: z.string() });

function actionOf(path: string, body: unknown): AuthAction | null {
  switch (path) {
    case "/sign-up/email": {
      const parsed = emailBodySchema.safeParse(body);
      return parsed.success
        ? { kind: "signUp", email: parsed.data.email }
        : null;
    }
    case "/sign-in/email": {
      const parsed = emailBodySchema.safeParse(body);
      return parsed.success
        ? { kind: "signIn", email: parsed.data.email }
        : null;
    }
    case "/organization/create": {
      const parsed = organizationBodySchema.safeParse(body);
      return parsed.success
        ? { kind: "createOrganization", name: parsed.data.name }
        : null;
    }
    case "/organization/invite-member": {
      const parsed = emailBodySchema.safeParse(body);
      return parsed.success
        ? { kind: "createInvitation", email: parsed.data.email }
        : null;
    }
    default:
      return null;
  }
}

export function authRulesHook(rules: readonly AuthRule[]) {
  return createAuthMiddleware(async (ctx) => {
    const action = actionOf(ctx.path, ctx.body);
    if (action === null) return;
    for (const rule of rules) await rule(action);
  });
}
