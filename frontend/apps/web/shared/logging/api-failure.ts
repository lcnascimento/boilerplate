import { logger } from "./logger";

import type { Result } from "@boilerplate/api/core";

export async function logAPIFailure<T>(
  operation: string,
  call: Promise<Result<T>>
): Promise<Result<T>> {
  const result = await call;
  if (result.kind === "error" && result.error.status >= 500) {
    const { status, code, message, retryable } = result.error;
    logger.error(
      { operation, status, code, message, retryable },
      "API request failed"
    );
  }
  return result;
}
