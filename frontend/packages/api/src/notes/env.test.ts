import { describe, expect, it } from "vitest";

import { parseNotesEnv } from "./index";

describe("parseNotesEnv", () => {
  it("reads the notes service URL, without a trailing slash", () => {
    expect(
      parseNotesEnv({ NOTES_API_URL: "http://localhost:8080/", PATH: "/bin" })
    ).toEqual({ NOTES_API_URL: "http://localhost:8080" });
  });

  it.each([
    [{}, "Invalid environment: NOTES_API_URL is required"],
    [
      { NOTES_API_URL: "" },
      "Invalid environment: NOTES_API_URL must be an http(s) URL",
    ],
    [
      { NOTES_API_URL: "localhost:8080" },
      "Invalid environment: NOTES_API_URL must be an http(s) URL",
    ],
  ])("rejects %j with a clear message", (env, message) => {
    expect(() => parseNotesEnv(env)).toThrow(message);
  });
});
