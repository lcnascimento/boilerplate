export const SIDEBAR_COOKIE_NAME = "sidebar_state";

export function parseSidebarOpen(value: unknown): boolean {
  return value !== "false";
}
