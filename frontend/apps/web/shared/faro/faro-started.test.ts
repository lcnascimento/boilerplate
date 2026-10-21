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
import { renderHook } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { reportBoundaryError } from "./faro";
import { useReportBoundaryError } from "./use-report-boundary-error";

class MemoryTransport extends BaseTransport {
  readonly name = "memory";
  readonly version = "0.0.0";
  items: TransportItem[] = [];

  send(items: TransportItem | TransportItem[]): void {
    this.items.push(...[items].flat());
  }

  exceptions() {
    return this.items
      .filter((item) => item.type === TransportItemType.EXCEPTION)
      .map(({ payload }) => payload);
  }
}

const transport = new MemoryTransport();

beforeAll(() => {
  initializeFaro({
    app: { name: "web" },
    instrumentations: [],
    transports: [transport],
    batching: { enabled: false },
    sessionTracking: { enabled: false },
  });
});

beforeEach(() => {
  transport.items = [];
});

describe("reportBoundaryError", () => {
  it("pushes the error and its digest to Faro", () => {
    const pushError = vi.spyOn(faro.api, "pushError");
    const error = Object.assign(new Error("boom"), { digest: "1234567890" });

    reportBoundaryError(error);

    expect(pushError).toHaveBeenCalledWith(error, {
      context: { digest: "1234567890" },
    });
    expect(transport.exceptions()).toMatchObject([
      { type: "Error", value: "boom", context: { digest: "1234567890" } },
    ]);
  });
});

describe("useReportBoundaryError", () => {
  it("reports each error the boundary receives once", () => {
    const boom = new Error("boom");
    const { rerender } = renderHook(
      ({ error }) => {
        useReportBoundaryError(error);
      },
      { initialProps: { error: boom } }
    );

    rerender({ error: boom });
    rerender({ error: new Error("bang") });

    expect(transport.exceptions()).toMatchObject([
      { type: "Error", value: "boom" },
      { type: "Error", value: "bang" },
    ]);
  });
});
