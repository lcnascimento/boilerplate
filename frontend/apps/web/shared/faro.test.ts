/**
 * @vitest-environment jsdom
 */
import {
  BaseTransport,
  faro,
  initializeFaro,
  type TransportItem,
  TransportItemType,
} from "@grafana/faro-web-sdk";
import { describe, expect, it, vi } from "vitest";

import { parseFaroEnv, reportBoundaryError } from "./faro";

class MemoryTransport extends BaseTransport {
  readonly name = "memory";
  readonly version = "0.0.0";
  readonly items: TransportItem[] = [];

  send(items: TransportItem | TransportItem[]): void {
    this.items.push(...[items].flat());
  }
}

function exceptionsIn(transport: MemoryTransport) {
  return transport.items
    .filter((item) => item.type === TransportItemType.EXCEPTION)
    .map(({ payload }) => payload);
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

// Faro is a process-wide singleton: the no-op case runs before the one that starts it.
describe("reportBoundaryError", () => {
  const transport = new MemoryTransport();

  it("does nothing while Faro is off", () => {
    expect(() =>
      reportBoundaryError(new Error("before Faro started"))
    ).not.toThrow();
  });

  it("pushes the error and its digest to Faro", () => {
    initializeFaro({
      app: { name: "web" },
      instrumentations: [],
      transports: [transport],
      batching: { enabled: false },
      sessionTracking: { enabled: false },
    });
    const pushError = vi.spyOn(faro.api, "pushError");
    const error = Object.assign(new Error("boom"), { digest: "1234567890" });

    reportBoundaryError(error);

    expect(pushError).toHaveBeenCalledWith(error, {
      context: { digest: "1234567890" },
    });
    expect(exceptionsIn(transport)).toMatchObject([
      { type: "Error", value: "boom", context: { digest: "1234567890" } },
    ]);
  });
});
