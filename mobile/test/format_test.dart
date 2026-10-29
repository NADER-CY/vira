import 'package:flutter_test/flutter_test.dart';
import 'package:vira/core/format/jalali.dart';
import 'package:vira/core/format/persian.dart';

void main() {
  group('digits', () {
    test('ASCII ↔ Persian / Arabic-Indic', () {
      expect(toPersianDigits('09121234567'), '۰۹۱۲۱۲۳۴۵۶۷');
      expect(toLatinDigits('۰۹۱۲٠١٢'), '0912012');
    });
  });

  group('money (integer Rial → Toman)', () {
    test('groups with the Persian separator', () {
      expect(formatNumber(1234567), '۱٬۲۳۴٬۵۶۷');
      expect(formatToman(125000), '۱۲٬۵۰۰ تومان');
    });
    test('never rounds silently', () => expect(formatToman(125005), '۱۲٬۵۰۰٫۵ تومان'));
    test('zero', () => expect(formatToman(0), '۰ تومان'));
  });

  group('Jalali calendar (verified against ICU)', () {
    final cases = {
      (2026, 10, 30): const JalaliDate(1405, 8, 8),
      (2025, 3, 21): const JalaliDate(1404, 1, 1),
      (2024, 3, 20): const JalaliDate(1403, 1, 1),
      (2025, 3, 20): const JalaliDate(1403, 12, 30),
      (2026, 3, 20): const JalaliDate(1404, 12, 29),
      (1990, 1, 1): const JalaliDate(1368, 10, 11),
      (2000, 2, 29): const JalaliDate(1378, 12, 10),
    };
    cases.forEach((g, j) {
      test('$g → $j and back', () {
        expect(JalaliDate.fromGregorian(g.$1, g.$2, g.$3), j);
        expect(j.toGregorian(), g);
      });
    });
    test('leap years', () {
      expect(JalaliDate.isLeapYear(1403), isTrue);
      expect(JalaliDate.isLeapYear(1404), isFalse);
    });
    test('formatted in Tehran time', () {
      // 21:00 UTC on 29 Oct is 00:30 on 30 Oct in Tehran → 8 Aban.
      expect(formatJalaliDate(DateTime.utc(2026, 10, 29, 21)), '۸ آبان ۱۴۰۵');
    });
  });

  group('Iranian mobile numbers', () {
    test('normalises every common form', () {
      for (final s in ['09121234567', '+989121234567', '00989121234567', '9121234567', '۰۹۱۲ ۱۲۳ ۴۵۶۷', '0912-123-4567']) {
        expect(normalizeIranMobile(s), '09121234567', reason: s);
      }
    });
    test('rejects invalid input', () {
      for (final s in ['0912123456', '08121234567', 'abc', '']) {
        expect(normalizeIranMobile(s), isNull, reason: s);
      }
    });
    test('display', () => expect(formatIranMobile('09121234567'), '\u2066۰۹۱۲ ۱۲۳ ۴۵۶۷\u2069'));
  });

  test('relative time', () {
    final now = DateTime(2026, 10, 29, 12);
    expect(formatRelative(now.subtract(const Duration(minutes: 3)), now: now), '۳ دقیقه پیش');
    expect(formatRelative(now.subtract(const Duration(days: 2)), now: now), '۲ روز پیش');
  });
}
