import { metrics, type Counter } from "@opentelemetry/api";
import { createAuthMiddleware, isAPIError } from "better-auth/api";

export const LOGIN_FAILURES_METRIC = "iam.login.failures";

// A metric label, so the set stays closed.
const reasons = [
  "invalid_credentials",
  "rate_limited",
  "email_taken",
  "invalid_input",
] as const;

type LoginFailureReason = (typeof reasons)[number];

const loginPaths = ["/sign-in/", "/sign-up/"];

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

// Created on first use, because the meter provider exists only once instrumentation.node.ts has started the SDK.
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

export const loginFailureHook = createAuthMiddleware((ctx) => {
  const returned = ctx.context.returned;
  if (isLoginPath(ctx.path) && isAPIError(returned)) {
    record(reasonOf(returned.statusCode, returned.body?.code));
  }
  return Promise.resolve();
});

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
