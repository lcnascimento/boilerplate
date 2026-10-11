import { getTranslations } from "next-intl/server";

import type { User } from "@boilerplate/types";

/**
 * The only user while the app has no authentication, named in the request
 * locale. This is the seam for a future auth: return the session user here and
 * the shell keeps working.
 */
export async function getLocalUser(): Promise<User> {
  const t = await getTranslations("nav");
  return { id: "local", name: t("localUser") };
}
