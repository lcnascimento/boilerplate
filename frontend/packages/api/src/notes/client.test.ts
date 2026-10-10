import {
  createServer,
  type IncomingMessage,
  type ServerResponse,
} from "node:http";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { createNotesClient, noteIdSchema } from "./index";

type Route = (req: IncomingMessage, body: string, res: ServerResponse) => void;

function json(res: ServerResponse, status: number, payload: string) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(payload);
}

const ID = "note_0199c3a1-7b2e-7c4d-9e5f-0a1b2c3d4e5f";
const OTHER_ID = "note_0199c3a0-0000-7000-8000-000000000000";
const MISSING_ID = "note_0199c3a1-ffff-7fff-bfff-ffffffffffff";

// Payloads as the Go service writes them (see openapi/notes.yaml).
const routes: Record<string, Route> = {
  "GET /notes/v1/notes": (_req, _body, res) => {
    json(
      res,
      200,
      `{"notes":[{"id":"${ID}","title":"Compras","body":"leite","created_at":"2026-10-10T12:30:00.123456Z","revision":3},{"id":"${OTHER_ID}","title":"Ideias","body":"","created_at":"2026-10-09T08:00:00-03:00"}]}`
    );
  },
  [`GET /notes/v1/notes/${ID}`]: (_req, _body, res) => {
    json(
      res,
      200,
      `{"id":"${ID}","title":"Compras","body":"leite","created_at":"2026-10-10T12:30:00Z"}`
    );
  },
  [`GET /notes/v1/notes/${OTHER_ID}`]: (_req, _body, res) => {
    // Out of contract: `created_at` is not RFC 3339.
    json(
      res,
      200,
      `{"id":"${OTHER_ID}","title":"Ideias","body":"","created_at":"ontem"}`
    );
  },
  [`GET /notes/v1/notes/${MISSING_ID}`]: (_req, _body, res) => {
    json(
      res,
      404,
      '{"code":"NOTE_NOT_FOUND","message":"failed to get note","retryable":false,"details":{"reasons":["note not found"]}}'
    );
  },
  "POST /notes/v1/notes": (req, body, res) => {
    const sent: unknown = JSON.parse(body);
    if (
      typeof sent === "object" &&
      sent !== null &&
      "title" in sent &&
      sent.title === ""
    ) {
      json(
        res,
        400,
        '{"code":"INVALID_INPUT","message":"invalid input","retryable":false,"details":{"reasons":["[title] is a required field"]}}'
      );
      return;
    }
    // Echo what arrived in `body` so the test sees the wire payload.
    json(
      res,
      201,
      JSON.stringify({
        id: ID,
        title: "Compras",
        body: `${req.headers["content-type"] ?? ""} ${body}`,
        created_at: "2026-10-10T12:30:00Z",
      })
    );
  },
  [`DELETE /notes/v1/notes/${ID}`]: (_req, _body, res) => {
    res.writeHead(204);
    res.end();
  },
  [`DELETE /notes/v1/notes/${MISSING_ID}`]: (_req, _body, res) => {
    json(
      res,
      404,
      '{"code":"NOTE_NOT_FOUND","message":"failed to delete note","retryable":false,"details":{"reasons":["note not found"]}}'
    );
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

// Ids reach the client already parsed at the app's boundary.
const noteId = (value: string) => noteIdSchema.parse(value);

describe("notes client", () => {
  it("lists notes in camelCase with created_at as a Date, ignoring unknown fields", async () => {
    const result = await createNotesClient({ baseUrl }).list();

    expect(result).toEqual({
      kind: "ok",
      data: [
        {
          id: ID,
          title: "Compras",
          body: "leite",
          createdAt: new Date("2026-10-10T12:30:00.123Z"),
        },
        {
          id: OTHER_ID,
          title: "Ideias",
          body: "",
          createdAt: new Date("2026-10-09T11:00:00Z"),
        },
      ],
    });
  });

  it("gets one note", async () => {
    const result = await createNotesClient({ baseUrl }).get(noteId(ID));

    expect(result).toEqual({
      kind: "ok",
      data: {
        id: ID,
        title: "Compras",
        body: "leite",
        createdAt: new Date("2026-10-10T12:30:00Z"),
      },
    });
  });

  it("turns a note outside the schema into UNEXPECTED", async () => {
    const result = await createNotesClient({ baseUrl }).get(noteId(OTHER_ID));

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

  it("returns the service's 404 for an unknown note", async () => {
    const result = await createNotesClient({ baseUrl }).get(noteId(MISSING_ID));

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "NOTE_NOT_FOUND",
        message: "failed to get note",
        retryable: false,
        details: { reasons: ["note not found"] },
        status: 404,
      },
    });
  });

  it("creates a note, sending the wire shape as JSON", async () => {
    const result = await createNotesClient({ baseUrl }).create({
      title: "Compras",
      body: "leite",
    });

    expect(result).toEqual({
      kind: "ok",
      data: {
        id: ID,
        title: "Compras",
        body: 'application/json {"title":"Compras","body":"leite"}',
        createdAt: new Date("2026-10-10T12:30:00Z"),
      },
    });
  });

  it("omits an absent body from the wire payload", async () => {
    const result = await createNotesClient({ baseUrl }).create({
      title: "Compras",
    });

    expect(result).toMatchObject({
      kind: "ok",
      data: { body: 'application/json {"title":"Compras"}' },
    });
  });

  it("returns the service's 400 with its reasons", async () => {
    const result = await createNotesClient({ baseUrl }).create({ title: "" });

    expect(result).toEqual({
      kind: "error",
      error: {
        code: "INVALID_INPUT",
        message: "invalid input",
        retryable: false,
        details: { reasons: ["[title] is a required field"] },
        status: 400,
      },
    });
  });

  it("deletes a note", async () => {
    const result = await createNotesClient({ baseUrl }).delete(noteId(ID));

    expect(result).toEqual({ kind: "ok", data: undefined });
  });

  it("returns the service's 404 when deleting an unknown note", async () => {
    const result = await createNotesClient({ baseUrl }).delete(
      noteId(MISSING_ID)
    );

    expect(result).toMatchObject({
      kind: "error",
      error: { code: "NOTE_NOT_FOUND", status: 404 },
    });
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

    const result = await createNotesClient({
      baseUrl: `http://127.0.0.1:${String(address.port)}`,
    }).list();

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
});
