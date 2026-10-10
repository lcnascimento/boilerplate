import { describe, expect, it } from "vitest";

import {
  defaultLocale,
  isLocale,
  LOCALE_COOKIE_NAME,
  locales,
  parseLocale,
} from "./config";

describe("locales", () => {
  it("supports pt-BR and en, with pt-BR as the default", () => {
    expect(locales).toEqual(["pt-BR", "en"]);
    expect(defaultLocale).toBe("pt-BR");
  });

  it("stores the locale in the boilerplate-locale cookie", () => {
    expect(LOCALE_COOKIE_NAME).toBe("boilerplate-locale");
  });
});

describe("isLocale", () => {
  it.each([
    ["pt-BR", true],
    ["en", true],
    ["pt-br", false],
    ["fr", false],
    ["", false],
    [undefined, false],
    [42, false],
  ])("isLocale(%j) is %s", (value, expected) => {
    expect(isLocale(value)).toBe(expected);
  });
});

describe("parseLocale", () => {
  it.each([
    ["en", "en"],
    ["pt-BR", "pt-BR"],
    ["fr", "pt-BR"],
    ["", "pt-BR"],
    [undefined, "pt-BR"],
    [null, "pt-BR"],
  ])("parseLocale(%j) returns %s", (value, expected) => {
    expect(parseLocale(value)).toBe(expected);
  });
});
