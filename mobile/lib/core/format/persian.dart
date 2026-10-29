/// Persian text and number formatting (doc 09 §3). Money is integer Rial in data and Toman on screen (B-1).
library;

import 'jalali.dart';

const _persianDigits = '۰۱۲۳۴۵۶۷۸۹';
const _arabicDigits = '٠١٢٣٤٥٦٧٨٩';

/// Replaces ASCII digits with Persian digits.
String toPersianDigits(Object value) {
  final s = value.toString();
  final b = StringBuffer();
  for (final r in s.runes) {
    b.write(r >= 0x30 && r <= 0x39 ? _persianDigits[r - 0x30] : String.fromCharCode(r));
  }
  return b.toString();
}

/// Normalises Persian and Arabic-Indic digits to ASCII (input parsing: phone numbers, OTP codes).
String toLatinDigits(String value) {
  final b = StringBuffer();
  for (final ch in value.split('')) {
    final p = _persianDigits.indexOf(ch);
    final a = _arabicDigits.indexOf(ch);
    b.write(p >= 0 ? '$p' : (a >= 0 ? '$a' : ch));
  }
  return b.toString();
}

/// Groups an integer in threes with the Persian thousands separator (U+066C) and Persian digits.
String formatNumber(int n) {
  final neg = n < 0;
  final digits = n.abs().toString();
  final b = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) b.write('٬');
    b.write(digits[i]);
  }
  return '${neg ? '\u2066-\u2069' : ''}${toPersianDigits(b.toString())}';
}

/// Formats integer Rial as Toman ("۱۲٬۵۰۰ تومان"). Never rounds silently: a fractional Toman is shown.
String formatToman(int rial) {
  final whole = rial ~/ 10;
  final frac = (rial % 10).abs();
  final s = formatNumber(whole) + (frac != 0 ? '٫${toPersianDigits(frac)}' : '');
  return '$s تومان';
}

/// Parses an Iranian mobile number typed with any digits, with or without +98/0098/0, into 09xxxxxxxxx.
/// Returns null when the input is not a valid mobile number.
String? normalizeIranMobile(String input) {
  var s = toLatinDigits(input).replaceAll(RegExp(r'[\s\-()]'), '');
  if (s.startsWith('+98')) {
    s = '0${s.substring(3)}';
  } else if (s.startsWith('0098')) {
    s = '0${s.substring(4)}';
  } else if (s.startsWith('9') && s.length == 10) {
    s = '0$s';
  }
  return RegExp(r'^09\d{9}$').hasMatch(s) ? s : null;
}

/// Displays a mobile number as ۰۹۱۲ ۳۴۵ ۶۷۸۹ inside an LTR isolate.
String formatIranMobile(String normalized) =>
    '\u2066${toPersianDigits('${normalized.substring(0, 4)} ${normalized.substring(4, 7)} ${normalized.substring(7)}')}\u2069';

/// Jalali date in Persian, e.g. "۸ آبان ۱۴۰۵". [d] is converted to Tehran time (UTC+03:30, no DST since 2022).
String formatJalaliDate(DateTime d) {
  final tehran = d.toUtc().add(const Duration(hours: 3, minutes: 30));
  final j = JalaliDate.fromGregorian(tehran.year, tehran.month, tehran.day);
  return '${toPersianDigits(j.day)} ${j.monthName} ${toPersianDigits(j.year)}';
}

/// Relative time in Persian ("۳ دقیقه پیش").
String formatRelative(DateTime d, {DateTime? now}) {
  final diff = (now ?? DateTime.now()).difference(d);
  final past = !diff.isNegative;
  final a = diff.abs();
  String v;
  if (a.inSeconds < 60) {
    return past ? 'لحظاتی پیش' : 'چند لحظه دیگر';
  } else if (a.inMinutes < 60) {
    v = '${toPersianDigits(a.inMinutes)} دقیقه';
  } else if (a.inHours < 24) {
    v = '${toPersianDigits(a.inHours)} ساعت';
  } else {
    v = '${toPersianDigits(a.inDays)} روز';
  }
  return past ? '$v پیش' : '$v دیگر';
}

/// Wraps a left-to-right fragment (usernames, codes, Latin product names) in first-strong isolates.
String isolate(String s) => '\u2068$s\u2069';
