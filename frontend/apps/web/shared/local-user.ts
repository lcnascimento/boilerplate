import { getTranslations } from "next-intl/server";

import type { User } from "@boilerplate/types";

export async function getLocalUser(): Promise<User> {
  const t = await getTranslations("nav");
  return { id: "local", name: t("localUser") };
}
