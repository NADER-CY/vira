import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers.dart';

void main() {
  testWidgets('the whole app is right-to-left Persian', (tester) async {
    await setPhone(tester);
    await pumpApp(tester);
    final ctx = tester.element(find.text('خانه'));
    expect(Directionality.of(ctx), TextDirection.rtl);
    expect(Localizations.localeOf(ctx).languageCode, 'fa');
    // RTL: Home (first destination) is to the right of Profile (last).
    expect(tester.getCenter(find.text('خانه')).dx, greaterThan(tester.getCenter(find.text('پروفایل')).dx));
  });

  testWidgets('home shows the live server state from GET /v1/config', (tester) async {
    await setPhone(tester);
    await pumpApp(tester);
    expect(find.text('اتصال به سرور ویرا برقرار است'), findsOneWidget);
    expect(find.textContaining('۸ قابلیت مالی'), findsOneWidget);
  });

  testWidgets('server unreachable → error state with retry', (tester) async {
    await setPhone(tester);
    await pumpApp(tester, serverUp: false);
    expect(find.textContaining('اتصال به سرور برقرار نشد'), findsOneWidget);
    expect(find.text('تلاش دوباره'), findsOneWidget);
  });

  testWidgets('tabs navigate; create sheet lists items with their phases', (tester) async {
    await setPhone(tester);
    await pumpApp(tester);
    await tester.tap(find.text('ریلز'));
    await tester.pumpAndSettle();
    expect(find.text('ریلز و ویدیوهای کوتاه'), findsOneWidget);
    await tester.tap(find.bySemanticsLabel('ایجاد'));
    await tester.pumpAndSettle();
    expect(find.text('پخش زنده'), findsOneWidget);
    expect(find.text('فاز ۸'), findsOneWidget);
  });

  testWidgets('theme switch: light ↔ dark from the profile, persisted', (tester) async {
    await setPhone(tester);
    await pumpApp(tester, prefs: {'vira.theme': 'light'}, location: '/profile');
    BuildContext ctx() => tester.element(find.text('ظاهر'));
    expect(Theme.of(ctx()).brightness, Brightness.light);
    await tester.tap(find.text('تیره'));
    await tester.pumpAndSettle();
    expect(Theme.of(ctx()).brightness, Brightness.dark);
    expect(Theme.of(ctx()).scaffoldBackgroundColor, const Color(0xFF0B0F17));
  });

  testWidgets('wallet is shown locked behind gate G1', (tester) async {
    await setPhone(tester);
    await pumpApp(tester, location: '/profile');
    await tester.scrollUntilVisible(find.text('کیف پول'), 200);
    await tester.tap(find.text('کیف پول'));
    await tester.pumpAndSettle();
    expect(find.textContaining('دروازهٔ G1'), findsOneWidget);
  });
}
