import { describe, expect, it } from "vitest";

import { createNoteInputSchema, noteIdSchema } from "./index";

function issues(input: unknown) {
  const result = createNoteInputSchema.safeParse(input);
  return result.success
    ? []
    : result.error.issues.map(({ path, code }) => ({ path, code }));
}

describe("createNoteInputSchema", () => {
  it("accepts a title with an optional body, trimming the title", () => {
    expect(createNoteInputSchema.parse({ title: "  Compras " })).toEqual({
      title: "Compras",
    });
    expect(
      createNoteInputSchema.parse({ title: "Compras", body: "leite" })
    ).toEqual({ title: "Compras", body: "leite" });
  });

  it.each([
    [{}, [{ path: ["title"], code: "invalid_type" }]],
    [{ title: "   " }, [{ path: ["title"], code: "too_small" }]],
    [{ title: "a".repeat(121) }, [{ path: ["title"], code: "too_big" }]],
    [
      { title: "ok", body: "a".repeat(10_001) },
      [{ path: ["body"], code: "too_big" }],
    ],
  ])("rejects %j", (input, expected) => {
    expect(issues(input)).toEqual(expected);
  });

  it("accepts the limits themselves", () => {
    expect(
      issues({ title: "a".repeat(120), body: "a".repeat(10_000) })
    ).toEqual([]);
  });
});

describe("noteIdSchema", () => {
  it.each([
    ["note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f", true],
    ["0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f", false],
    ["note_123", false],
  ])("%s is valid: %s", (id, valid) => {
    expect(noteIdSchema.safeParse(id).success).toBe(valid);
  });
});
