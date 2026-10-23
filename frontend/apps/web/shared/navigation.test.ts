import { describe, expect, it } from "vitest";

import { breadcrumbTrail, isActiveHref, routesIn } from "./navigation";

describe("routesIn", () => {
  it("keeps only the product content in the main menu", () => {
    expect(routesIn("product").map(({ href }) => href)).toEqual([
      "/",
      // example:begin
      "/notes",
      // example:end
    ]);
  });

  it("puts the Organization settings in the Organization menu", () => {
    expect(routesIn("organization").map(({ href }) => href)).toEqual([
      "/settings/organization",
      "/settings/members",
    ]);
  });

  it("puts the account settings in the User menu", () => {
    expect(routesIn("account").map(({ href }) => href)).toEqual([
      "/settings/preferences",
    ]);
  });
});

describe("breadcrumbTrail", () => {
  it("shows only the current home crumb on /", () => {
    expect(breadcrumbTrail("/")).toEqual([
      { href: "/", labelKey: "nav.home", current: true },
    ]);
  });

  it("links home and marks the matched route as current", () => {
    expect(breadcrumbTrail("/settings/preferences")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      {
        href: "/settings/preferences",
        labelKey: "nav.preferences",
        current: true,
      },
    ]);
  });

  it("ignores a trailing slash", () => {
    expect(breadcrumbTrail("/settings/preferences/")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      {
        href: "/settings/preferences",
        labelKey: "nav.preferences",
        current: true,
      },
    ]);
  });

  it("keeps a route as a link below a path it does not own", () => {
    expect(breadcrumbTrail("/settings/preferences/extra")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      {
        href: "/settings/preferences",
        labelKey: "nav.preferences",
        current: false,
      },
    ]);
  });

  it("names the Organization settings page", () => {
    expect(breadcrumbTrail("/settings/organization")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      {
        href: "/settings/organization",
        labelKey: "nav.organizationSettings",
        current: true,
      },
    ]);
  });

  it("names the Members settings page", () => {
    expect(breadcrumbTrail("/settings/members")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      { href: "/settings/members", labelKey: "members.title", current: true },
    ]);
  });

  // example:begin
  it("links a feature route from its detail pages", () => {
    expect(breadcrumbTrail("/notes/note_1")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
      { href: "/notes", labelKey: "notes.title", current: false },
    ]);
  });
  // example:end

  it("falls back to the home link on an unknown route", () => {
    expect(breadcrumbTrail("/does-not-exist")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
    ]);
  });

  it("does not match a route by a partial segment", () => {
    expect(breadcrumbTrail("/settings/preferencesX")).toEqual([
      { href: "/", labelKey: "nav.home", current: false },
    ]);
  });
});

describe("isActiveHref", () => {
  it.each([
    { pathname: "/", href: "/", active: true },
    { pathname: "/settings/preferences", href: "/", active: false },
    {
      pathname: "/settings/preferences",
      href: "/settings/preferences",
      active: true,
    },
    {
      pathname: "/settings/preferences/x",
      href: "/settings/preferences",
      active: true,
    },
    {
      pathname: "/settings/preferencesX",
      href: "/settings/preferences",
      active: false,
    },
  ])("$pathname under $href is $active", ({ pathname, href, active }) => {
    expect(isActiveHref(pathname, href)).toBe(active);
  });
});
