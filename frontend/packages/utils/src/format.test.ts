import { describe, expect, it } from "vitest";

import { formatCurrency, formatDate, formatRelativeTime } from "./format";

describe("formatCurrency", () => {
  it.each([
    [1234.56, {}, "R$ 1.234,56"],
    [0, {}, "R$ 0,00"],
    [-100, {}, "-R$ 100,00"],
    [1234.56, { currency: "USD", locale: "en-US" }, "$1,234.56"],
  ])("formatCurrency(%d, %j) is %j", (value, options, expected) => {
    expect(formatCurrency(value, options)).toBe(expected);
  });
});

describe("formatDate", () => {
  const date = new Date("2024-01-15T12:00:00Z");

  it("uses a long pt-BR date by default", () => {
    expect(formatDate(date, { timeZone: "UTC" })).toBe("15 de janeiro de 2024");
  });

  it("formats in the given locale", () => {
    expect(formatDate(date, { locale: "en-US", timeZone: "UTC" })).toBe(
      "January 15, 2024"
    );
  });

  it("formats in the given time zone", () => {
    const lateNight = new Date("2024-01-15T01:00:00Z");
    expect(formatDate(lateNight, { timeZone: "America/Sao_Paulo" })).toBe(
      "14 de janeiro de 2024"
    );
  });
});

describe("formatRelativeTime", () => {
  const now = new Date("2024-01-15T12:00:00Z");
  const ago = (seconds: number) => new Date(now.getTime() - seconds * 1000);

  it.each([
    [0, "pt-BR", "agora"],
    [30, "pt-BR", "há 30 segundos"],
    [3600, "pt-BR", "há 1 hora"],
    [2 * 86400, "pt-BR", "anteontem"],
    [3 * 86400, "pt-BR", "há 3 dias"],
    [-3600, "pt-BR", "em 1 hora"],
    [3600, "en-US", "1 hour ago"],
    [86400, "en-US", "yesterday"],
  ])("%ds ago in %s is %j", (seconds, locale, expected) => {
    expect(formatRelativeTime(ago(seconds), { locale, now })).toBe(expected);
  });
});
