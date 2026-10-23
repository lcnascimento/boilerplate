import { describe, expect, it } from "vitest";

import { organizationSlug } from "./organization-slug";

describe("organizationSlug", () => {
  it.each([
    ["Acme", "acme-x7k2p9"],
    ["  Açaí & Cia. Ltda  ", "acai-cia-ltda-x7k2p9"],
    ["São João -- Padaria", "sao-joao-padaria-x7k2p9"],
    ["日本", "org-x7k2p9"],
    ["a".repeat(80), `${"a".repeat(40)}-x7k2p9`],
  ])("derives %j into %j", (name, expected) => {
    expect(organizationSlug(name, () => "x7k2p9")).toBe(expected);
  });

  it("adds a different random suffix each time by default", () => {
    const first = organizationSlug("Acme");
    const second = organizationSlug("Acme");

    expect(first).toMatch(/^acme-[a-z0-9]{6}$/);
    expect(second).toMatch(/^acme-[a-z0-9]{6}$/);
    expect(first).not.toBe(second);
  });
});
