import { describe, expect, it } from "vitest";

import { parseSidebarOpen } from "./sidebar";

describe("parseSidebarOpen", () => {
  it.each([
    { value: "true", open: true },
    { value: "false", open: false },
    { value: undefined, open: true },
    { value: "garbage", open: true },
  ])("cookie $value opens the sidebar: $open", ({ value, open }) => {
    expect(parseSidebarOpen(value)).toBe(open);
  });
});
