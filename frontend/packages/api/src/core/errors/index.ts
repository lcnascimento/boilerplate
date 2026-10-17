import type { APIError } from "../result";
import type { z } from "zod";

export const ErrorCode = {
  Unavailable: "UNAVAILABLE",
  Unexpected: "UNEXPECTED",
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
