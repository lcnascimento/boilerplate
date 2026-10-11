const DEFAULT_LOCALE = "pt-BR";

export function formatCurrency(
  value: number,
  { currency = "BRL", locale = DEFAULT_LOCALE } = {}
): string {
  return new Intl.NumberFormat(locale, { style: "currency", currency }).format(
    value
  );
}

/** Long date, e.g. "15 de janeiro de 2024". */
export function formatDate(
  date: Date,
  {
    locale = DEFAULT_LOCALE,
    timeZone,
  }: { locale?: string; timeZone?: string } = {}
): string {
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "long",
    day: "numeric",
    timeZone,
  }).format(date);
}

const RELATIVE_UNITS: readonly {
  unit: Intl.RelativeTimeFormatUnit;
  seconds: number;
}[] = [
  { unit: "year", seconds: 31_536_000 },
  { unit: "month", seconds: 2_592_000 },
  { unit: "week", seconds: 604_800 },
  { unit: "day", seconds: 86_400 },
  { unit: "hour", seconds: 3_600 },
  { unit: "minute", seconds: 60 },
  { unit: "second", seconds: 1 },
];

/** Time relative to `now`, e.g. "há 1 hora" or "em 2 dias". */
export function formatRelativeTime(
  date: Date,
  {
    locale = DEFAULT_LOCALE,
    now = new Date(),
  }: { locale?: string; now?: Date } = {}
): string {
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const diffInSeconds = Math.round((date.getTime() - now.getTime()) / 1000);

  for (const { unit, seconds } of RELATIVE_UNITS) {
    const count = Math.trunc(diffInSeconds / seconds);
    if (count !== 0) return rtf.format(count, unit);
  }

  return rtf.format(0, "second");
}
