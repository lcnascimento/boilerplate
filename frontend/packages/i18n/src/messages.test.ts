import { describe, expect, it } from "vitest";

import { messages } from "./messages";

function keyPaths(value: unknown, prefix = ""): string[] {
  if (typeof value !== "object" || value === null) return [prefix];
  return Object.entries(value).flatMap(([key, child]) =>
    keyPaths(child, prefix ? `${prefix}.${key}` : key)
  );
}

describe("messages", () => {
  it("ships the common, errors and validation namespaces", () => {
    expect(Object.keys(messages["pt-BR"])).toEqual([
      "common",
      "errors",
      "validation",
    ]);
  });

  it("translates the same keys in every locale", () => {
    expect(keyPaths(messages.en)).toEqual(keyPaths(messages["pt-BR"]));
  });

  it.each([
    ["pt-BR", "Excluir", "Campo obrigatório"],
    ["en", "Delete", "Required field"],
  ] as const)("%s has its own texts", (locale, deleteLabel, required) => {
    expect(messages[locale].common.delete).toBe(deleteLabel);
    expect(messages[locale].validation.required).toBe(required);
  });
});
