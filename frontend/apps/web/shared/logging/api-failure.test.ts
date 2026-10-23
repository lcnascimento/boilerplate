import { beforeEach, describe, expect, it, vi } from "vitest";

import { logAPIFailure } from "./api-failure";
import { logger } from "./logger";

import type { Result } from "@boilerplate/api/core";

vi.mock("./logger", () => ({ logger: { error: vi.fn() } }));

const error = vi.mocked(logger.error);

describe("logAPIFailure", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("logs a call the service could not answer", async () => {
    const result: Result<string> = {
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    };

    await expect(
      logAPIFailure("items.list", Promise.resolve(result))
    ).resolves.toBe(result);

    expect(error).toHaveBeenCalledExactlyOnceWith(
      {
        operation: "items.list",
        status: 503,
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
      },
      "API request failed"
    );
  });

  it("logs a 5xx error the service returned", async () => {
    await logAPIFailure(
      "items.get",
      Promise.resolve({
        kind: "error",
        error: {
          code: "ERR_DB_DOWN",
          message: "database unavailable",
          retryable: true,
          status: 500,
        },
      })
    );

    expect(error).toHaveBeenCalledExactlyOnceWith(
      {
        operation: "items.get",
        status: 500,
        code: "ERR_DB_DOWN",
        message: "database unavailable",
        retryable: true,
      },
      "API request failed"
    );
  });

  it.each<[string, Result<string>]>([
    ["a success", { kind: "ok", data: "item" }],
    [
      "a 4xx the caller handles",
      {
        kind: "error",
        error: {
          code: "ERR_NOT_FOUND",
          message: "item not found",
          retryable: false,
          status: 404,
        },
      },
    ],
  ])("does not log %s", async (_name, result) => {
    await expect(
      logAPIFailure("items.get", Promise.resolve(result))
    ).resolves.toBe(result);

    expect(error).not.toHaveBeenCalled();
  });
});
