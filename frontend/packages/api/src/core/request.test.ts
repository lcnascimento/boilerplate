import {
  createServer,
  type IncomingMessage,
  type ServerResponse,
} from "node:http";

import {
  afterAll,
  afterEach,
  beforeAll,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { z } from "zod";

import { logger } from "./logger";

import { request } from "./index";

type Route = (req: IncomingMessage, body: string, res: ServerResponse) => void;

function json(res: ServerResponse, status: number, payload: string) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(payload);
}

const routes: Record<string, Route> = {
  "GET /widgets/1": (_req, _body, res) => {
    json(res, 200, '{"id":"w_1","display_name":"Gear"}');
  },
  "GET /widgets/wrong-shape": (_req, _body, res) => {
    json(res, 200, '{"id":1,"name":"Gear"}');
  },
  "GET /widgets/not-json": (_req, _body, res) => {
    res.writeHead(200, { "Content-Type": "text/html" });
    res.end("<html>proxy page</html>");
  },
  "GET /widgets/missing": (_req, _body, res) => {
    json(
      res,
      404,
      '{"code":"ERR_WIDGET_NOT_FOUND","message":"widget not found","retryable":false}'
    );
  },
  "GET /widgets/invalid": (_req, _body, res) => {
    json(
      res,
      400,
      '{"code":"ERR_INVALID_INPUT","message":"invalid input","retryable":false,"details":{"reasons":["title is required"]}}'
    );
  },
  "POST /widgets": (req, body, res) => {
    json(
      res,
      201,
      JSON.stringify({
        id: "w_2",
        display_name: `${req.headers["content-type"] ?? ""} ${body}`,
      })
    );
  },
  "DELETE /widgets/1": (_req, _body, res) => {
    res.writeHead(204);
    res.end();
  },
  "GET /widgets/down": (_req, _body, res) => {
    json(
      res,
      503,
      '{"code":"ERR_DB_DOWN","message":"database unavailable","retryable":true}'
    );
  },
  "GET /widgets/crash": (_req, _body, res) => {
    res.writeHead(500, { "Content-Type": "text/plain" });
    res.end("Internal Server Error");
  },
};

const server = createServer((req, res) => {
  let body = "";
  req.on("data", (chunk: Buffer) => (body += chunk.toString()));
  req.on("end", () => {
    const route = routes[`${req.method ?? ""} ${req.url ?? ""}`];
    if (route) route(req, body, res);
    else json(res, 599, "{}");
  });
});

let baseUrl = "";

beforeAll(async () => {
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (address === null || typeof address === "string") {
    throw new Error("test server is not listening on a TCP port");
  }
  baseUrl = `http://127.0.0.1:${String(address.port)}`;
});

afterAll(async () => {
  await new Promise((resolve) => server.close(resolve));
});

const widgetSchema = z
  .object({ id: z.string(), display_name: z.string() })
  .transform(({ id, display_name }) => ({ id, displayName: display_name }));

describe("request", () => {
  it("returns the schema output of a successful response", async () => {
    const result = await request({
      url: `${baseUrl}/widgets/1`,
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "ok",
      data: { id: "w_1", displayName: "Gear" },
    });
  });

  it("sends the body as JSON with the given method", async () => {
    const result = await request({
      url: `${baseUrl}/widgets`,
      method: "POST",
      body: { display_name: "Gear" },
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "ok",
      data: {
        id: "w_2",
        displayName: 'application/json {"display_name":"Gear"}',
      },
    });
  });

  it("accepts an empty 204 response", async () => {
    const result = await request({
      url: `${baseUrl}/widgets/1`,
      method: "DELETE",
      schema: z.undefined(),
    });

    expect(result).toEqual({ kind: "ok", data: undefined });
  });

  it("turns a network failure into UNAVAILABLE without throwing", async () => {
    const closed = createServer();
    await new Promise<void>((resolve) =>
      closed.listen(0, "127.0.0.1", resolve)
    );
    const address = closed.address();
    await new Promise((resolve) => closed.close(resolve));
    if (address === null || typeof address === "string") {
      throw new Error("closed server had no TCP port");
    }

    const result = await request({
      url: `http://127.0.0.1:${String(address.port)}/widgets/1`,
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "UNAVAILABLE",
        message: "service unavailable",
        retryable: true,
        status: 503,
      },
    });
  });

  it.each([["/widgets/wrong-shape"], ["/widgets/not-json"]])(
    "turns a success body outside the schema (%s) into UNEXPECTED",
    async (path) => {
      const result = await request({
        url: `${baseUrl}${path}`,
        schema: widgetSchema,
      });

      expect(result).toEqual({
        kind: "error",
        error: {
          code: "UNEXPECTED",
          message: "unexpected response from service",
          retryable: false,
          status: 502,
        },
      });
    }
  );

  it("returns the service's APIError with the HTTP status", async () => {
    const result = await request({
      url: `${baseUrl}/widgets/missing`,
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "ERR_WIDGET_NOT_FOUND",
        message: "widget not found",
        retryable: false,
        status: 404,
      },
    });
  });

  it("keeps the reasons the service sends in details", async () => {
    const result = await request({
      url: `${baseUrl}/widgets/invalid`,
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "ERR_INVALID_INPUT",
        message: "invalid input",
        retryable: false,
        details: { reasons: ["title is required"] },
        status: 400,
      },
    });
  });

  it("turns an error body that is not an APIError into UNEXPECTED", async () => {
    const result = await request({
      url: `${baseUrl}/widgets/crash`,
      schema: widgetSchema,
    });

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "UNEXPECTED",
        message: "unexpected response from service",
        retryable: false,
        status: 502,
      },
    });
  });
});

describe("request logging", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("logs a network failure with its cause", async () => {
    const error = vi.spyOn(logger, "error");
    const closed = createServer();
    await new Promise<void>((resolve) =>
      closed.listen(0, "127.0.0.1", resolve)
    );
    const address = closed.address();
    await new Promise((resolve) => closed.close(resolve));
    if (address === null || typeof address === "string") {
      throw new Error("closed server had no TCP port");
    }
    const url = `http://127.0.0.1:${String(address.port)}/widgets/1`;

    await request({ url, schema: widgetSchema });

    expect(error).toHaveBeenCalledExactlyOnceWith(
      {
        err: new TypeError("fetch failed"),
        method: "GET",
        url,
        code: "UNAVAILABLE",
      },
      "API request failed"
    );
  });

  it.each([
    ["/widgets/down", 503, "ERR_DB_DOWN"],
    ["/widgets/crash", 500, "UNEXPECTED"],
    ["/widgets/not-json", 200, "UNEXPECTED"],
    ["/widgets/wrong-shape", 200, "UNEXPECTED"],
  ])("logs %s as a failure", async (path, status, code) => {
    const error = vi.spyOn(logger, "error");

    await request({ url: `${baseUrl}${path}`, schema: widgetSchema });

    expect(error).toHaveBeenCalledExactlyOnceWith(
      { method: "GET", url: `${baseUrl}${path}`, status, code },
      "API request failed"
    );
  });

  it.each([["/widgets/1"], ["/widgets/missing"], ["/widgets/invalid"]])(
    "does not log %s, which the caller handles",
    async (path) => {
      const error = vi.spyOn(logger, "error");

      await request({ url: `${baseUrl}${path}`, schema: widgetSchema });

      expect(error).not.toHaveBeenCalled();
    }
  );
});
