import type { User } from "@boilerplate/types";

/**
 * The only user while the app has no authentication. This is the seam for a
 * future auth: replace it with the session user and the shell keeps working.
 */
export const LOCAL_USER: User = {
  id: "local",
  name: "Usuário local",
};
