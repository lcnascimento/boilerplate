/**
 * @vitest-environment jsdom
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

import { parseFaroEnv } from "./faro";

import type { TransportItem } from "@grafana/faro-web-sdk";

async function freshFaro() {
  vi.resetModules();
  const sdk = await import("@grafana/faro-web-sdk");
  const { reportBoundaryError } = await import("./faro");

  class MemoryTransport extends sdk.BaseTransport {
    readonly name = "memory";
    readonly version = "0.0.0";
    readonly items: TransportItem[] = [];

    send(items: TransportItem | TransportItem[]): void {
      this.items.push(...[items].flat());
    }

    exceptions() {
      return this.items
        .filter((item) => item.type === sdk.TransportItemType.EXCEPTION)
        .map(({ payload }) => payload);
    }
  }

  return { sdk, reportBoundaryError, transport: new MemoryTransport() };
}

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
  let faro: Awaited<ReturnType<typeof freshFaro>>;

  beforeEach(async () => {
    faro = await freshFaro();
  });

  it("does nothing while Faro is off", () => {
    expect(() =>
      faro.reportBoundaryError(new Error("before Faro started"))
    ).not.toThrow();
  });

  it("pushes the error and its digest to Faro", () => {
    faro.sdk.initializeFaro({
      app: { name: "web" },
      instrumentations: [],
      transports: [faro.transport],
      batching: { enabled: false },
      sessionTracking: { enabled: false },
    });
    const pushError = vi.spyOn(faro.sdk.faro.api, "pushError");
    const error = Object.assign(new Error("boom"), { digest: "1234567890" });

    faro.reportBoundaryError(error);

    expect(pushError).toHaveBeenCalledWith(error, {
      context: { digest: "1234567890" },
    });
    expect(faro.transport.exceptions()).toMatchObject([
      { type: "Error", value: "boom", context: { digest: "1234567890" } },
    ]);
  });
});
