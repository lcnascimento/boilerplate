import { describe, expect, it } from "vitest";

import { canNote } from "./can-note";

describe("canNote", () => {
  it.each([
    ["owner", "create", true],
    ["owner", "delete", true],
    ["admin", "create", true],
    ["admin", "delete", false],
    ["viewer", "create", false],
    ["viewer", "delete", false],
    [null, "create", false],
    [null, "delete", false],
  ] as const)("lets a %s %s notes: %s", (role, action, expected) => {
    expect(canNote(role, action)).toBe(expected);
  });
});
