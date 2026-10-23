import { describe, expect, it } from "vitest";

import { parseTtl } from "./ttl";

describe("parseTtl", () => {
  it.each([
    ["90s", 90],
    ["5m", 300],
    ["1h", 3600],
    ["12h", 43_200],
  ])("reads %s as %i seconds", (raw, seconds) => {
    expect(parseTtl(raw)).toBe(seconds);
  });

  it.each([["0m"], ["1d"], ["h"], ["5"], ["-1h"], [" 1h"]])(
    "rejects %j",
    (raw) => {
      expect(() => parseTtl(raw)).toThrow(
        `--ttl must be a positive number of s, m or h (e.g. 1h), got "${raw}"`
      );
    }
  );
});
