/**
 * @vitest-environment jsdom
 */
import { faro } from "@grafana/faro-web-sdk";
import { describe, expect, it } from "vitest";

import { parseFaroEnv, reportBoundaryError } from "./faro";

describe("parseFaroEnv", () => {
  it.each([
    ["no URL", {}],
    ["a blank URL", { NEXT_PUBLIC_FARO_URL: "" }],
  ])("leaves Faro off with %s", (_, env) => {
    expect(parseFaroEnv(env)).toEqual({ kind: "disabled" });
  });

  it("samples every session by default", () => {
    expect(
      parseFaroEnv({ NEXT_PUBLIC_FARO_URL: "http://localhost:12347/collect" })
    ).toEqual({
      kind: "enabled",
      url: "http://localhost:12347/collect",
      sampleRate: 1,
    });
  });

  it("samples the fraction NEXT_PUBLIC_FARO_SAMPLE_RATE asks for", () => {
    expect(
      parseFaroEnv({
        NEXT_PUBLIC_FARO_URL: "http://localhost:12347/collect",
        NEXT_PUBLIC_FARO_SAMPLE_RATE: "0.25",
      })
    ).toEqual({
      kind: "enabled",
      url: "http://localhost:12347/collect",
      sampleRate: 0.25,
    });
  });

  it("rejects a sample rate outside 0..1", () => {
    expect(() =>
      parseFaroEnv({
        NEXT_PUBLIC_FARO_URL: "http://localhost:12347/collect",
        NEXT_PUBLIC_FARO_SAMPLE_RATE: "2",
      })
    ).toThrow();
  });
});

describe("reportBoundaryError", () => {
  it("does nothing while Faro is off", () => {
    expect(() =>
      reportBoundaryError(new Error("before Faro started"))
    ).not.toThrow();
    expect(faro.config).toBeUndefined();
  });
});
