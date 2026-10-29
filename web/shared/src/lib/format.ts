// Persian formatting helpers (doc 09 §3). Money is stored as integer Rial and displayed in Toman (B-1).

const PERSIAN = "۰۱۲۳۴۵۶۷۸۹";
const ARABIC_INDIC = "٠١٢٣٤٥٦٧٨٩";

/** Replaces ASCII digits with Persian digits. */
export function toPersianDigits(value: string | number | bigint): string {
  return String(value).replace(/[0-9]/g, (d) => PERSIAN[Number(d)]!);
}

/** Normalises Persian and Arabic-Indic digits to ASCII (for input parsing: phone numbers, codes). */
export function toLatinDigits(value: string): string {
  return value.replace(/[۰-۹٠-٩]/g, (d) => {
    const p = PERSIAN.indexOf(d);
    return String(p >= 0 ? p : ARABIC_INDIC.indexOf(d));
  });
}

const groupFmt = new Intl.NumberFormat("fa-IR", { maximumFractionDigits: 0 });

/** Formats an integer with Persian digits and Persian thousands separators. */
export function formatNumber(n: number | bigint): string {
  return groupFmt.format(n);
}

/**
 * Formats an integer Rial amount as Toman ("۱۲٬۵۰۰ تومان"). Never rounds silently (doc 09 §3):
 * an amount that is not a whole number of Toman is shown with its decimal.
 */
export function formatToman(rial: number | bigint): string {
  const r = BigInt(rial);
  const neg = r < 0n;
  const abs = neg ? -r : r;
  const whole = abs / 10n;
  const frac = abs % 10n;
  let s = groupFmt.format(whole);
  if (frac !== 0n) s += "٫" + toPersianDigits(frac.toString());
  return `${neg ? "⁦-⁩" : ""}${s} تومان`;
}

const jalaliDate = new Intl.DateTimeFormat("fa-IR-u-ca-persian", { year: "numeric", month: "long", day: "numeric" });
const jalaliDateTime = new Intl.DateTimeFormat("fa-IR-u-ca-persian", {
  year: "numeric", month: "long", day: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "Asia/Tehran",
});

/** Jalali (Solar Hijri) date for display, e.g. "۸ آبان ۱۴۰۵". Data stays Gregorian UTC. */
export function formatJalaliDate(d: Date | string): string {
  return jalaliDate.format(typeof d === "string" ? new Date(d) : d);
}

/** Jalali date and Tehran time. */
export function formatJalaliDateTime(d: Date | string): string {
  return jalaliDateTime.format(typeof d === "string" ? new Date(d) : d);
}

const rtf = new Intl.RelativeTimeFormat("fa-IR", { numeric: "auto" });

/** Relative time in Persian ("۳ دقیقه پیش"). */
export function formatRelative(d: Date | string, now: Date = new Date()): string {
  const t = typeof d === "string" ? new Date(d) : d;
  const sec = Math.round((t.getTime() - now.getTime()) / 1000);
  const abs = Math.abs(sec);
  if (abs < 60) return rtf.format(sec, "second");
  if (abs < 3600) return rtf.format(Math.round(sec / 60), "minute");
  if (abs < 86400) return rtf.format(Math.round(sec / 3600), "hour");
  return rtf.format(Math.round(sec / 86400), "day");
}

/** Wraps a left-to-right fragment (IDs, e-mail, codes, Latin names) in Unicode isolates (FSI…PDI). */
export function isolate(s: string): string {
  return `⁨${s}⁩`;
}
