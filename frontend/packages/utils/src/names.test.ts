import { describe, expect, it } from "vitest";

import { getInitials } from "./names";

describe("getInitials", () => {
  it.each([
    ["Ada Lovelace", "AL"],
    ["Ada King Lovelace", "AL"],
    ["ada lovelace", "AL"],
    ["  Ada   Lovelace  ", "AL"],
    ["José García", "JG"],
    ["Usuário local", "UL"],
    ["Ada", "AD"],
    ["A", "A"],
    ["", "?"],
    ["   ", "?"],
  ])("getInitials(%j) is %j", (name, expected) => {
    expect(getInitials(name)).toBe(expected);
  });
});
