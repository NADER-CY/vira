import { describe, expect, it } from "vitest";
import { formatJalaliDate, formatNumber, formatToman, isolate, toLatinDigits, toPersianDigits } from "./format";

describe("Persian formatting", () => {
  it("converts digits both ways", () => {
    expect(toPersianDigits("09121234567")).toBe("۰۹۱۲۱۲۳۴۵۶۷");
    expect(toLatinDigits("۰۹۱۲٠١٢")).toBe("0912012");
  });

  it("groups numbers with Persian digits", () => {
    expect(formatNumber(1234567)).toBe("۱٬۲۳۴٬۵۶۷");
  });

  it("formats integer Rial as Toman without silent rounding", () => {
    expect(formatToman(125_000)).toBe("۱۲٬۵۰۰ تومان");
    expect(formatToman(125_005)).toBe("۱۲٬۵۰۰٫۵ تومان");
    expect(formatToman(0n)).toBe("۰ تومان");
    expect(formatToman(90_071_992_547_409_910n)).toContain("تومان");
  });

  it("formats dates in the Jalali calendar", () => {
    // 2026-10-30 is 8 Aban 1405.
    expect(formatJalaliDate("2026-10-30T12:00:00Z")).toBe("۸ آبان ۱۴۰۵");
  });

  it("isolates LTR fragments", () => {
    expect(isolate("ORD-1")).toBe("⁨ORD-1⁩");
  });
});
