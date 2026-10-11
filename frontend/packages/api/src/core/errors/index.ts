import type { APIError } from "../result";
import type { z } from "zod";

// Isomorphic: client components import this entry (`@boilerplate/api/core/errors`)
// to branch on error codes; `request()` lives in the server-only `@boilerplate/api/core`.

/** Codes of the errors this package builds itself instead of reading them from a service. */
export const ErrorCode = {
  /** The service could not be reached. */
  Unavailable: "UNAVAILABLE",
  /** The service answered outside its schema. */
  Unexpected: "UNEXPECTED",
  /** The input failed a local schema; same code as go-kit's validation error. */
  InvalidInput: "INVALID_INPUT",
} as const;

export const unavailableError: APIError = {
  code: ErrorCode.Unavailable,
  message: "service unavailable",
  retryable: true,
  status: 503,
};

export const unexpectedError: APIError = {
  code: ErrorCode.Unexpected,
  message: "unexpected response from service",
  retryable: false,
  status: 502,
};

/** A local Zod failure in the shape of the service's 400, one reason per issue. */
export function invalidInputError(error: z.ZodError): APIError {
  return {
    code: ErrorCode.InvalidInput,
    message: "invalid input",
    retryable: false,
    details: {
      reasons: error.issues.map((issue) =>
        issue.path.length === 0
          ? issue.message
          : `[${issue.path.join(".")}] ${issue.message}`
      ),
    },
    status: 400,
  };
}
