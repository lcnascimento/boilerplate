import { describe, expect, it } from "vitest";

import { breadcrumbTrail, isActiveHref } from "./navigation";

describe("breadcrumbTrail", () => {
  it("shows only the current home crumb on /", () => {
    expect(breadcrumbTrail("/")).toEqual([
      { href: "/", labelKey: "home", current: true },
    ]);
  });

  it("links home and marks the matched route as current", () => {
    expect(breadcrumbTrail("/profile/preferences")).toEqual([
      { href: "/", labelKey: "home", current: false },
      { href: "/profile/preferences", labelKey: "preferences", current: true },
    ]);
  });

  it("ignores a trailing slash", () => {
    expect(breadcrumbTrail("/profile/preferences/")).toEqual([
      { href: "/", labelKey: "home", current: false },
      { href: "/profile/preferences", labelKey: "preferences", current: true },
    ]);
  });

  it("keeps a route as a link below a path it does not own", () => {
    expect(breadcrumbTrail("/profile/preferences/extra")).toEqual([
      { href: "/", labelKey: "home", current: false },
      { href: "/profile/preferences", labelKey: "preferences", current: false },
    ]);
  });

  it("falls back to the home link on an unknown route", () => {
    expect(breadcrumbTrail("/does-not-exist")).toEqual([
      { href: "/", labelKey: "home", current: false },
    ]);
  });

  it("does not match a route by a partial segment", () => {
    expect(breadcrumbTrail("/profile/preferencesX")).toEqual([
      { href: "/", labelKey: "home", current: false },
    ]);
  });
});

describe("isActiveHref", () => {
  it.each([
    { pathname: "/", href: "/", active: true },
    { pathname: "/profile/preferences", href: "/", active: false },
    {
      pathname: "/profile/preferences",
      href: "/profile/preferences",
      active: true,
    },
    {
      pathname: "/profile/preferences/x",
      href: "/profile/preferences",
      active: true,
    },
    {
      pathname: "/profile/preferencesX",
      href: "/profile/preferences",
      active: false,
    },
  ])("$pathname under $href is $active", ({ pathname, href, active }) => {
    expect(isActiveHref(pathname, href)).toBe(active);
  });
});
