import { House, SlidersHorizontal, type LucideIcon } from "lucide-react";

import type { AppMessages } from "@/shared/_messages";
import type { Route } from "next";

/**
 * A message key qualified by its namespace (`nav.home`, `notes.title`), so a
 * feature labels its route with its own namespace.
 */
type NavLabelKey = {
  [N in keyof AppMessages]: `${N}.${Extract<keyof AppMessages[N], string>}`;
}[keyof AppMessages];

interface NavRoute {
  href: Route;
  labelKey: NavLabelKey;
  icon: LucideIcon;
  /** Where the route is listed: the sidebar or the user menu. */
  area: "sidebar" | "userMenu";
}

/**
 * The single navigation config. The sidebar, the user menu and the breadcrumb
 * all read from it, so they can't disagree.
 */
export const navRoutes = [
  { href: "/", labelKey: "nav.home", icon: House, area: "sidebar" },
  {
    href: "/profile/preferences",
    labelKey: "nav.preferences",
    icon: SlidersHorizontal,
    area: "userMenu",
  },
] as const satisfies readonly NavRoute[];

export function routesIn(area: NavRoute["area"]): NavRoute[] {
  return navRoutes.filter((route) => route.area === area);
}

/** Whether `pathname` is `href` or a page below it. `/` matches only itself. */
export function isActiveHref(pathname: string, href: string): boolean {
  const path = trimTrailingSlash(pathname);
  if (href === "/") return path === "/";
  return path === href || path.startsWith(`${href}/`);
}

export interface Crumb {
  href: Route;
  labelKey: NavLabelKey;
  /** The crumb is the page being shown, not a link. */
  current: boolean;
}

/**
 * The breadcrumb for `pathname`: home, then every route that contains the
 * path, outermost first. An unknown path keeps only the home link.
 */
export function breadcrumbTrail(pathname: string): Crumb[] {
  const path = trimTrailingSlash(pathname);
  const home: Crumb = {
    href: "/",
    labelKey: "nav.home",
    current: path === "/",
  };
  const matched = navRoutes
    .filter((route) => route.href !== "/" && isActiveHref(path, route.href))
    .sort((a, b) => a.href.length - b.href.length)
    .map((route): Crumb => ({
      href: route.href,
      labelKey: route.labelKey,
      current: route.href === path,
    }));
  return [home, ...matched];
}

function trimTrailingSlash(pathname: string): string {
  return pathname.length > 1 && pathname.endsWith("/")
    ? pathname.slice(0, -1)
    : pathname;
}
