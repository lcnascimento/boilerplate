import path from "node:path";

export function storageStateOf(email: string): string {
  return path.join(import.meta.dirname, ".auth", `${email}.json`);
}

export const signedOut = { cookies: [], origins: [] };
