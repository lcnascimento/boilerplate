import { SpanStatusCode, trace } from "@opentelemetry/api";

import { logger } from "./logger";

import type { Instrumentation } from "next";

export const recordRequestError: Instrumentation.onRequestError = (
  error,
  request,
  context
) => {
  const err = error instanceof Error ? error : new Error(String(error));
  const digest =
    typeof error === "object" && error !== null && "digest" in error
      ? String(error.digest)
      : undefined;

  trace
    .getActiveSpan()
    ?.setStatus({ code: SpanStatusCode.ERROR, message: err.message });

  logger.error(
    {
      err,
      digest,
      method: request.method,
      path: request.path,
      routePath: context.routePath,
      routeType: context.routeType,
    },
    "request failed"
  );
};
