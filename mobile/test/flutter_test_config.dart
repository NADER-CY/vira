import 'dart:async';
import 'dart:io';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

/// Loads the real fonts (Vazirmatn + Material Icons) so golden images show actual Persian text and icons
/// instead of the test font's boxes.
Future<void> testExecutable(FutureOr<void> Function() testMain) async {
  TestWidgetsFlutterBinding.ensureInitialized();
  final vazir = FontLoader('Vazirmatn');
  for (final w in ['Regular', 'Medium', 'SemiBold', 'Bold', 'ExtraBold']) {
    vazir.addFont(rootBundle.load('assets/fonts/Vazirmatn-$w.ttf'));
  }
  await vazir.load();
  final root = Platform.environment['FLUTTER_ROOT'];
  final icons = File('$root/bin/cache/artifacts/material_fonts/MaterialIcons-Regular.otf');
  if (root != null && icons.existsSync()) {
    final loader = FontLoader('MaterialIcons')..addFont(icons.readAsBytes().then((b) => ByteData.sublistView(b)));
    await loader.load();
  }
  await testMain();
}
