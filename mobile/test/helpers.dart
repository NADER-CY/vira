import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vira/app.dart';
import 'package:vira/core/api/api_client.dart';
import 'package:vira/core/providers.dart';

const configJson = {
  'api_version': 'v1',
  'flags': {'app.maintenance_mode': false, 'seller.web_portal': false},
  'regulated_locked': {
    'payments.real_money': 'FG-1b', 'settlement.live': 'FG-1b', 'payouts.withdrawals': 'FG-1b', 'wallet.topup': 'G1',
    'usdt.deposits': 'G2', 'usdt.withdrawals': 'G2', 'usdt.conversion': 'G2', 'credit.bnpl': 'G3',
  },
  'config': <String, Object>{},
};

/// Pumps the real app with a fake HTTP backend (no network) and in-memory preferences.
Future<void> pumpApp(WidgetTester tester, {bool serverUp = true, Map<String, Object> prefs = const {}, String location = '/home'}) async {
  SharedPreferences.setMockInitialValues(prefs);
  final sp = await SharedPreferences.getInstance();
  final client = MockClient((req) async {
    if (!serverUp) throw http.ClientException('connection refused');
    if (req.url.path == '/v1/config') {
      return http.Response.bytes(utf8.encode(jsonEncode(configJson)), 200, headers: {'content-type': 'application/json'});
    }
    return http.Response('{"code":"not_found","status":404,"title":"یافت نشد"}', 404);
  });
  await tester.pumpWidget(ProviderScope(
    retry: (_, _) => null,
    overrides: [
      sharedPreferencesProvider.overrideWithValue(sp),
      apiClientProvider.overrideWithValue(ApiClient(baseUrl: 'http://test', client: client)),
    ],
    child: ViraApp(initialLocation: location),
  ));
  await tester.pumpAndSettle();
}

Future<void> setPhone(WidgetTester tester) async {
  tester.view.physicalSize = const Size(1170, 2532);
  tester.view.devicePixelRatio = 3;
  addTearDown(tester.view.reset);
}
