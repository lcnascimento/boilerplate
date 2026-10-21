import { randomBytes } from "node:crypto";

const MAX_BASE_LENGTH = 40;
const SUFFIX_LENGTH = 6;
const SUFFIX_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789";
const FALLBACK_BASE = "org";

function randomSuffix(): string {
  return Array.from(
    randomBytes(SUFFIX_LENGTH),
    (byte) => SUFFIX_ALPHABET[byte % SUFFIX_ALPHABET.length]
  ).join("");
}

export function organizationSlug(
  name: string,
  suffix: () => string = randomSuffix
): string {
  const base = name
    .normalize("NFKD")
    .replace(/\p{Mark}/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .slice(0, MAX_BASE_LENGTH)
    .replace(/^-+|-+$/g, "");
  return `${base || FALLBACK_BASE}-${suffix()}`;
}
