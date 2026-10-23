import { metrics, type Counter } from "@opentelemetry/api";
import { createAuthMiddleware, isAPIError } from "better-auth/api";

export const LOGIN_FAILURES_METRIC = "iam.login.failures";

const reasons = [
  "invalid_credentials",
  "rate_limited",
  "email_taken",
  "invalid_input",
] as const;

type LoginFailureReason = (typeof reasons)[number];

const signInFailureReasons = ["invalid_credentials", "invalid_input"] as const;

export type SignInFailureReason = (typeof signInFailureReasons)[number];

const SIGN_IN_PATH = "/sign-in/";
const loginPaths = [SIGN_IN_PATH, "/sign-up/"];

const HTTP_BAD_REQUEST = 400;
const HTTP_TOO_MANY_REQUESTS = 429;
const HTTP_INTERNAL_SERVER_ERROR = 500;

const reasonsByCode: Record<string, LoginFailureReason> = {
  INVALID_EMAIL_OR_PASSWORD: "invalid_credentials",
  USER_ALREADY_EXISTS: "email_taken",
  USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL: "email_taken",
};

function isLoginPath(path: string): boolean {
  return loginPaths.some((prefix) => path.startsWith(prefix));
}

function reasonOf(
  status: number,
  code: string | undefined
): LoginFailureReason | null {
  if (status === HTTP_TOO_MANY_REQUESTS) return "rate_limited";
  if (status < HTTP_BAD_REQUEST || status >= HTTP_INTERNAL_SERVER_ERROR)
    return null;
  return (
    (code !== undefined ? reasonsByCode[code] : undefined) ?? "invalid_input"
  );
}

let counter: Counter | undefined;

// Lazy: the OTel API meter is a no-op until instrumentation.node.ts registers the SDK.
function loginFailures(): Counter {
  counter ??= metrics
    .getMeter("@boilerplate/web")
    .createCounter(LOGIN_FAILURES_METRIC, {
      description: "Number of sign-ins and sign-ups refused, by reason",
    });
  return counter;
}

export function seedLoginFailures(): void {
  for (const reason of reasons) loginFailures().add(0, { reason });
}

function record(reason: LoginFailureReason | null): void {
  if (reason !== null) loginFailures().add(1, { reason });
}

function isSignInFailureReason(
  reason: LoginFailureReason | null
): reason is SignInFailureReason {
  return signInFailureReasons.some((known) => known === reason);
}

export function loginFailureHook(
  recordSignInFailure: (reason: SignInFailureReason) => Promise<void>
) {
  return createAuthMiddleware(async (ctx) => {
    const returned = ctx.context.returned;
    if (!isLoginPath(ctx.path) || !isAPIError(returned)) return;

    const reason = reasonOf(returned.statusCode, returned.body?.code);
    record(reason);
    if (ctx.path.startsWith(SIGN_IN_PATH) && isSignInFailureReason(reason)) {
      await recordSignInFailure(reason);
    }
  });
}

// The rate limiter answers before any Better Auth hook runs, so its 429s are counted around the handler.
export function withLoginFailures(
  handler: (request: Request) => Promise<Response>,
  basePath = "/api/auth"
): (request: Request) => Promise<Response> {
  return async (request) => {
    const response = await handler(request);
    const path = new URL(request.url).pathname.slice(basePath.length);
    if (response.status === HTTP_TOO_MANY_REQUESTS && isLoginPath(path)) {
      record("rate_limited");
    }
    return response;
  };
}
