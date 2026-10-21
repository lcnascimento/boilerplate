const UNIT_SECONDS = new Map([
  ["s", 1],
  ["m", 60],
  ["h", 60 * 60],
]);

const TTL = /^([1-9]\d*)([smh])$/;

export function parseTtl(raw: string): number {
  const [, amount, unit = ""] = TTL.exec(raw) ?? [];
  const unitSeconds = UNIT_SECONDS.get(unit);
  if (amount === undefined || unitSeconds === undefined) {
    throw new Error(
      `--ttl must be a positive number of s, m or h (e.g. 1h), got "${raw}"`
    );
  }
  return Number(amount) * unitSeconds;
}
