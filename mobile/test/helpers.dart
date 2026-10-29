import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vira/app.dart';
import 'package:vira/core/api/api_client.dart';
import 'package:vira/core/auth/session.dart';
import 'package:vira/core/media/picker.dart';
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

const me = 'u-me';
const other = 'u-sara';

/// In-memory stand-in for the Phase 2 API (no network). Synthetic data only — no real people or numbers.
class FakeApi {
  FakeApi({this.posts = 2, this.feedError = false, this.serverUp = true, this.privateOther = false});
  int posts;
  bool feedError, serverUp, privateOther;
  final List<http.Request> requests = [];
  final Map<String, bool> liked = {};
  final List<Map<String, Object?>> comments = [];
  final List<Map<String, Object?>> reports = [];
  final List<String> requestsPending = ['u-req1', 'u-req2'];
  Map<String, Object?> privacy = {'is_private': false, 'comments_from': 'everyone', 'hide_like_counts': false, 'settings': <String, Object?>{}};
  String handle = 'nima.dev';
  String bio = 'عکاس و طراح؛ عاشق رنگ‌های گرم.';
  int profileVersion = 3;
  bool followingOther = false;
  int refreshCalls = 0;

  static Map<String, Object?> author(String id) => switch (id) {
        me => {'id': me, 'handle': 'nima.dev', 'display_name': 'نیما توسعه', 'is_private': false},
        other => {'id': other, 'handle': 'sara_style', 'display_name': 'سارا استایل', 'is_private': false},
        _ => {'id': id, 'handle': 'user_${id.substring(id.length - 4)}', 'display_name': 'کاربر ${id.substring(id.length - 4)}', 'is_private': false},
      };

  Map<String, Object?> post(int i) {
    final id = 'p$i';
    final mine = i.isOdd;
    final l = liked[id] ?? false;
    return {
      'id': id,
      'author': author(mine ? me : other),
      'caption': i == 0 ? 'ترکیب رنگ لیمویی با کرم برای روزهای گرم تابستون واقعاً بی‌نظیره! #استایل_تابستانه #ویرا_مد' : 'پست شماره $i در فید آزمایشی',
      'visibility': 'public',
      'media': [
        {'id': 'm${i}a', 'url': '/v1/media/m${i}a', 'width': 1080, 'height': 1350},
        if (i == 0) {'id': 'm${i}b', 'url': '/v1/media/m${i}b', 'width': 1080, 'height': 1350},
        if (i == 0) {'id': 'm${i}c', 'url': '/v1/media/m${i}c', 'width': 1080, 'height': 1350},
      ],
      'like_count': 3420 + (l ? 1 : 0),
      'comment_count': 124 + comments.length,
      'viewer_liked': l,
      'is_owner': mine,
      'comments_allowed': true,
      'published_at': DateTime.now().toUtc().subtract(const Duration(hours: 3)).toIso8601String(),
      'edited_at': null,
      'version': 1,
      'viewer_follows_author': mine || followingOther,
    };
  }

  Map<String, Object?> profile(String id) {
    final isMe = id == me;
    return {
      ...author(id),
      if (isMe) 'handle': handle,
      if (!isMe && privateOther) 'is_private': true,
      'bio': isMe ? bio : 'طراح لباس و استایلیست',
      'website': isMe ? 'nima.example' : '',
      'stats': {'posts': isMe ? 148 : 32, 'followers': 38500, 'following': 320},
      'relationship': {'is_self': isMe, 'following': !isMe && followingOther, 'requested': false, 'follows_you': false, 'blocking': false},
      'can_view_posts': isMe || !privateOther || followingOther,
      if (isMe) 'version': profileVersion,
    };
  }

  static http.Response json(Object body, [int status = 200, Map<String, String> headers = const {}]) =>
      http.Response.bytes(utf8.encode(jsonEncode(body)), status, headers: {'content-type': 'application/json', ...headers});

  static http.Response problem(int status, String code, String title, {List<Map<String, String>>? errors}) =>
      json({'type': 'about:blank', 'status': status, 'code': code, 'title': title, 'errors': ?errors}, status);

  Map<String, Object?> page(List<Map<String, Object?>> items, {String? next}) => {'data': items, 'next_cursor': next};

  Future<http.Response> serve(http.Request req) async {
    requests.add(req);
    if (!serverUp) throw http.ClientException('connection refused');
    final p = req.url.path;
    final m = req.method;
    Map<String, dynamic> body() => req.body.isEmpty ? {} : jsonDecode(req.body) as Map<String, dynamic>;
    if (p == '/v1/config') return json(configJson);
    if (p == '/v1/auth/otp' && m == 'POST') return json({'challenge_id': 'ch1', 'expires_in': 120, 'resend_after': 60, 'code_length': 6}, 202);
    if (p == '/v1/auth/otp/verify') {
      final b = body();
      if (b['code'] == '111111') return problem(422, 'otp_invalid', 'کد نادرست است.');
      if (b['code'] == '999999') return problem(429, 'otp_locked', 'قفل شد.');
      return json({'access_token': 'at', 'token_type': 'Bearer', 'expires_in': 900, 'refresh_token': 'rt', 'session_id': 's1', 'user_id': me, 'handle': handle, 'is_new': false});
    }
    if (p == '/v1/auth/token/refresh') {
      refreshCalls++;
      return json({'access_token': 'at2', 'token_type': 'Bearer', 'expires_in': 900, 'refresh_token': 'rt2', 'session_id': 's1'});
    }
    if (p == '/v1/auth/logout') return http.Response('', 204);
    if (p == '/v1/feed/home' || p == '/v1/feed/following') {
      if (feedError) return problem(503, 'unavailable', 'سرویس موقتاً در دسترس نیست.');
      return json(page([for (var i = 0; i < posts; i++) post(i)]));
    }
    if (p == '/v1/me/profile' && m == 'GET') return json(profile(me), 200, {'ETag': '"$profileVersion"'});
    if (p == '/v1/me/profile' && m == 'PATCH') {
      final b = body();
      if (b['handle'] == 'taken') return problem(422, 'validation_failed', 'خطا', errors: [{'field': 'handle', 'code': 'taken'}]);
      handle = b['handle'] as String? ?? handle;
      bio = b['bio'] as String? ?? bio;
      profileVersion++;
      return json(profile(me));
    }
    if (p == '/v1/me/account') return json({'user_id': me, 'status': 'active', 'phone_masked': '09*****1234', 'intends_to_sell': false, 'created_at': '2026-10-01T10:00:00Z'});
    if (p == '/v1/me/sessions') {
      return json({'data': [
        {'id': 's1', 'platform': 'android', 'model': 'Pixel 9', 'app_version': '0.2.0', 'created_at': '2026-10-20T10:00:00Z', 'last_used_at': '2026-10-28T10:00:00Z', 'current': true},
        {'id': 's2', 'platform': 'ios', 'model': 'iPhone', 'app_version': '0.2.0', 'created_at': '2026-10-10T10:00:00Z', 'last_used_at': '2026-10-27T10:00:00Z', 'current': false},
      ]});
    }
    if (p.startsWith('/v1/me/sessions/') && m == 'DELETE') return http.Response('', 204);
    if (p == '/v1/me/sessions/revoke-others') return http.Response('', 204);
    if (p == '/v1/me/deletion') return json({'status': 'deletion_pending', 'effective_at': '2026-11-28T10:00:00Z'}, 202);
    if (p == '/v1/me/privacy' && m == 'GET') return json(privacy);
    if (p == '/v1/me/privacy' && m == 'PUT') {
      privacy = body();
      return json(privacy);
    }
    if (p == '/v1/me/blocks') return json({'data': <Object>[]});
    if (p == '/v1/me/follow-requests') return json(page([for (final id in requestsPending) {...author(id), 'since': '2026-10-28T10:00:00Z', 'following': false}]));
    final fr = RegExp(r'^/v1/me/follow-requests/([^/]+)/(approve|decline)$').firstMatch(p);
    if (fr != null) {
      requestsPending.remove(fr.group(1));
      return http.Response('', 204);
    }
    final user = RegExp(r'^/v1/users/([^/]+)(/.*)?$').firstMatch(p);
    if (user != null) {
      final id = user.group(1)!;
      final rest = user.group(2) ?? '';
      if (id == 'search') return json({'data': [author(other)]});
      if (rest == '') return json(profile(id));
      if (rest == '/posts') return json(page([for (var i = 0; i < 4; i++) post(i)]));
      if (rest == '/followers' || rest == '/following') {
        return json(page([for (final a in [author(other), author('u-follower')]) {...a, 'since': '2026-10-28T10:00:00Z', 'following': a['id'] == other && followingOther}]));
      }
      if (rest == '/follow' && m == 'POST') {
        followingOther = true;
        return json({'state': privateOther ? 'requested' : 'following'});
      }
      if (rest == '/follow' && m == 'DELETE') {
        followingOther = false;
        return http.Response('', 204);
      }
      if (rest == '/block') return http.Response('', 204);
    }
    final post_ = RegExp(r'^/v1/posts/([^/]+)(/.*)?$').firstMatch(p);
    if (post_ != null) {
      final id = post_.group(1)!;
      final rest = post_.group(2) ?? '';
      final i = int.tryParse(id.substring(1)) ?? 0;
      if (rest == '' && m == 'GET') return json(post(i));
      if (rest == '/like') {
        liked[id] = m == 'PUT';
        return json({'liked': liked[id], 'like_count': 3420 + (liked[id]! ? 1 : 0)});
      }
      if (rest == '/comments' && m == 'GET') {
        return json(page([
          for (final c in comments.reversed) c,
          {'id': 'c1', 'post_id': id, 'author': author(other), 'body': 'کیفیت پارچه‌اش عالیه! سایزبندی چطوره؟', 'created_at': DateTime.now().toUtc().subtract(const Duration(minutes: 45)).toIso8601String(), 'can_delete': false, 'is_post_author': false},
          {'id': 'c2', 'post_id': id, 'author': author(me), 'body': 'سلام، جدول سایز در توضیحات هست.', 'created_at': DateTime.now().toUtc().subtract(const Duration(minutes: 32)).toIso8601String(), 'can_delete': true, 'is_post_author': true},
        ]));
      }
      if (rest == '/comments' && m == 'POST') {
        final c = {'id': 'c${comments.length + 10}', 'post_id': id, 'author': author(me), 'body': body()['body'], 'created_at': DateTime.now().toUtc().toIso8601String(), 'can_delete': true, 'is_post_author': false};
        comments.add(c);
        return json(c, 201);
      }
      if (m == 'DELETE') return http.Response('', 204);
    }
    if (p == '/v1/posts' && m == 'POST') {
      final b = body();
      return json({...post(1), 'id': 'p9', 'caption': b['caption'], 'visibility': b['visibility']}, 201);
    }
    if (p.startsWith('/v1/comments/') && m == 'DELETE') return http.Response('', 204);
    if (p == '/v1/media/images') return json({'id': 'up1', 'url': '/v1/media/up1', 'width': 100, 'height': 100, 'purpose': 'post'}, 201);
    if (p == '/v1/reports') {
      reports.add(body());
      return json({'id': 'r1', 'state': 'open'}, 201);
    }
    return problem(404, 'not_found', 'یافت نشد');
  }
}

class FakePicker implements PhotoPicker {
  @override
  Future<List<PickedImage>> pick({int max = 1}) async => [(bytes: Uint8List.fromList(_png), name: 'a.png')];
}

// 1×1 transparent PNG.
const _png = [
  0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
  0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01,
  0x00, 0x00, 0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
];

/// Pumps the real app against [FakeApi]. [signedIn] restores a stored refresh token, as after a restart.
Future<FakeApi> pumpApp(
  WidgetTester tester, {
  FakeApi? api,
  bool signedIn = true,
  bool serverUp = true,
  Map<String, Object> prefs = const {},
  String location = '/home',
}) async {
  final fake = api ?? FakeApi(serverUp: serverUp);
  SharedPreferences.setMockInitialValues({
    if (signedIn) 'vira.user_id': me,
    if (signedIn) 'vira.handle': 'nima.dev',
    ...prefs,
  });
  final sp = await SharedPreferences.getInstance();
  final client = ApiClient(baseUrl: 'http://test', client: MockClient(fake.serve));
  final session = Session(api: client, store: MemoryTokenStore({if (signedIn) 'vira.refresh_token': 'rt0'}), prefs: sp);
  await session.restore();
  await tester.pumpWidget(ProviderScope(
    retry: (_, _) => null,
    overrides: [
      sharedPreferencesProvider.overrideWithValue(sp),
      apiClientProvider.overrideWithValue(client),
      sessionProvider.overrideWithValue(session),
      imagePickerProvider.overrideWithValue(FakePicker()),
    ],
    child: ViraApp(initialLocation: location),
  ));
  await tester.pumpAndSettle();
  return fake;
}

Future<void> setPhone(WidgetTester tester) async {
  tester.view.physicalSize = const Size(1170, 2532);
  tester.view.devicePixelRatio = 3;
  addTearDown(tester.view.reset);
}
