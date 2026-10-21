const UNIT_SECONDS = { s: 1, m: 60, h: 60 * 60 } as const;

const TTL = /^([1-9]\d*)([smh])$/;

export function parseTtl(raw: string): number {
  const match = TTL.exec(raw);
  if (match === null) {
    throw new Error(
      `--ttl must be a positive number of s, m or h (e.g. 1h), got "${raw}"`
    );
  }
  const [, amount = "", unit = "s"] = match;
  return Number(amount) * UNIT_SECONDS[unit as keyof typeof UNIT_SECONDS];
}
