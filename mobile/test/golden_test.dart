@Tags(['golden'])
library;

import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers.dart';

/// Visual evidence and regression baseline for the app shell in both themes (RTL, Vazirmatn).
/// Anti-aliasing can differ by a few pixels between engine builds, so up to 0.5 % of pixels may differ.
void main() {
  setUpAll(() {
    final base = goldenFileComparator as LocalFileComparator;
    goldenFileComparator = _TolerantComparator(Uri.parse('${base.basedir}golden_test.dart'), 0.005);
  });

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

class _TolerantComparator extends LocalFileComparator {
  _TolerantComparator(super.testFile, this.tolerance);
  final double tolerance;

  @override
  Future<bool> compare(Uint8List imageBytes, Uri golden) async {
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
