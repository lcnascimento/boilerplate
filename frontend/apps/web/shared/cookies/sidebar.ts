/**
 * Written by `SidebarProvider` (`@boilerplate/ui/components/sidebar`) when the
 * sidebar opens or collapses; the server shell reads it for `defaultOpen`.
 */
export const SIDEBAR_COOKIE_NAME = "sidebar_state";

/** The sidebar starts open unless the cookie says it was collapsed. */
export function parseSidebarOpen(value: unknown): boolean {
  return value !== "false";
}
