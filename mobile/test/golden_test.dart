@Tags(['golden'])
library;

import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers.dart';

/// Screenshots of every Phase 2 screen in both themes (RTL, Vazirmatn, Material Symbols) against synthetic data.
/// They are the regression baseline and the "implementation" half of the reference comparison sheets
/// (docs/progress/P2-ui/). Anti-aliasing differs slightly between engine builds: up to 0.5 % of pixels may
/// differ. A missing baseline is written (and reported) instead of failing, so new screens get a first image.
const screens = <String, ({String location, bool signedIn})>{
  'welcome': (location: '/welcome', signedIn: false),
  'home': (location: '/home', signedIn: true),
  'explore': (location: '/explore', signedIn: true),
  'profile': (location: '/profile', signedIn: true),
  'user_profile': (location: '/u/$other', signedIn: true),
  'edit_profile': (location: '/profile/edit', signedIn: true),
  'followers': (location: '/u/$me/followers', signedIn: true),
  'post_detail': (location: '/post/p0', signedIn: true),
  'comments': (location: '/post/p0/comments', signedIn: true),
  'create_post': (location: '/post/new', signedIn: true),
  'activity': (location: '/activity', signedIn: true),
  'settings': (location: '/settings', signedIn: true),
  'privacy': (location: '/settings/privacy', signedIn: true),
  'security': (location: '/settings/security', signedIn: true),
  'account': (location: '/settings/account', signedIn: true),
};

void main() {
  setUpAll(() {
    final base = goldenFileComparator as LocalFileComparator;
    goldenFileComparator = _TolerantComparator(Uri.parse('${base.basedir}golden_test.dart'), 0.005);
  });

  for (final theme in ['light', 'dark']) {
    for (final e in screens.entries) {
      testWidgets('${e.key} ($theme)', (tester) async {
        await setPhone(tester);
        await pumpApp(tester, signedIn: e.value.signedIn, prefs: {'vira.theme': theme}, location: e.value.location);
        await tester.pump(const Duration(milliseconds: 300));
        await expectLater(find.byType(MaterialApp), matchesGoldenFile('goldens/${e.key}_$theme.png'));
      });
    }

    testWidgets('otp ($theme)', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, signedIn: false, prefs: {'vira.theme': theme}, location: '/welcome');
      await tester.enterText(find.byKey(const Key('phone')), '09123456789');
      for (final k in ['consent-terms', 'consent-privacy', 'consent-age']) {
        await tester.scrollUntilVisible(find.byKey(Key(k)), 200, scrollable: find.byType(Scrollable).first);
        await tester.tap(find.byKey(Key(k)));
      }
      await tester.scrollUntilVisible(find.byKey(const Key('welcome-submit')), 200, scrollable: find.byType(Scrollable).first);
      await tester.tap(find.byKey(const Key('welcome-submit')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byKey(const Key('otp-input')), '48');
      await tester.pumpAndSettle();
      await expectLater(find.byType(MaterialApp), matchesGoldenFile('goldens/otp_$theme.png'));
    });

    testWidgets('report ($theme)', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, prefs: {'vira.theme': theme}, location: '/post/p0');
      await tester.tap(find.bySemanticsLabel('گزینه‌ها').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('گزارش پست'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('reason-scam')));
      await tester.pumpAndSettle();
      await expectLater(find.byType(MaterialApp), matchesGoldenFile('goldens/report_$theme.png'));
    });
  }
}

class _TolerantComparator extends LocalFileComparator {
  _TolerantComparator(super.testFile, this.tolerance);
  final double tolerance;

  @override
  Future<bool> compare(Uint8List imageBytes, Uri golden) async {
    final file = File.fromUri(basedir.resolveUri(golden));
    if (!file.existsSync()) {
      file.createSync(recursive: true);
      file.writeAsBytesSync(imageBytes);
      // ignore: avoid_print
      print('NEW GOLDEN written: ${golden.path}');
      return true;
    }
    final result = await GoldenFileComparator.compareLists(imageBytes, await getGoldenBytes(golden));
    if (result.passed || result.diffPercent <= tolerance) {
      result.dispose();
      return true;
    }
    final error = await generateFailureOutput(result, golden, basedir);
    result.dispose();
    throw FlutterError(error);
  }
}
