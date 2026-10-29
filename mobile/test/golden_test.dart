@Tags(['golden'])
library;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers.dart';

/// Visual evidence and regression baseline for the app shell in both themes (RTL, Vazirmatn).
void main() {
  for (final theme in ['light', 'dark']) {
    for (final screen in ['home', 'profile']) {
      testWidgets('$screen ($theme)', (tester) async {
        await setPhone(tester);
        await pumpApp(tester, prefs: {'vira.theme': theme}, location: '/$screen');
        await expectLater(find.byType(MaterialApp), matchesGoldenFile('goldens/${screen}_$theme.png'));
      });
    }
  }
}
