import { describe, expect, it } from "vitest";

import {
  isAuthPath,
  parseCallbackUrl,
  SIGNED_IN_HOME,
  signInHrefFor,
} from "./routes";

describe("parseCallbackUrl", () => {
  it("keeps a path of this app, with its query", () => {
    expect(parseCallbackUrl("/profile/preferences?tab=2")).toBe(
      "/profile/preferences?tab=2"
    );
  });

  it.each([
    [null],
    [""],
    ["https://evil.test/notes"],
    ["//evil.test/notes"],
    ["/\\evil.test"],
    ["notes"],
  ])("sends %j to the signed-in home", (raw) => {
    expect(parseCallbackUrl(raw)).toBe(SIGNED_IN_HOME);
  });
});

describe("signInHrefFor", () => {
  it("carries the page to return to", () => {
    expect(signInHrefFor("/profile/preferences?tab=2")).toBe(
      "/sign-in?callbackUrl=%2Fprofile%2Fpreferences%3Ftab%3D2"
    );
  });
});

describe("isAuthPath", () => {
  it.each([
    ["/sign-in", true],
    ["/sign-up", true],
    ["/sign-up/extra", false],
    ["/notes", false],
  ])("says whether %j is an auth page", (pathname, expected) => {
    expect(isAuthPath(pathname)).toBe(expected);
  });
});
