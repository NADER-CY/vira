import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vira/core/api/api_client.dart';
import 'package:vira/core/api/models.dart';
import 'package:vira/core/auth/session.dart';

http.Response _json(Object b, [int s = 200]) => http.Response.bytes(utf8.encode(jsonEncode(b)), s, headers: {'content-type': 'application/json'});

Future<(Session, ApiClient, List<http.Request>)> _setup(Future<http.Response> Function(http.Request) h, {String? refresh = 'rt0'}) async {
  SharedPreferences.setMockInitialValues({});
  final reqs = <http.Request>[];
  final api = ApiClient(baseUrl: 'http://t', client: MockClient((r) {
    reqs.add(r);
    return h(r);
  }));
  final s = Session(api: api, store: MemoryTokenStore({'vira.refresh_token': ?refresh}), prefs: await SharedPreferences.getInstance());
  await s.restore();
  return (s, api, reqs);
}

void main() {
  test('expired access token: one refresh, then the call is retried with the new token', () async {
    var calls = 0;
    final (s, api, reqs) = await _setup((r) async {
      if (r.url.path == '/v1/auth/token/refresh') {
        calls++;
        return _json({'access_token': 'new$calls', 'refresh_token': 'rt$calls', 'session_id': 's'});
      }
      return r.headers['Authorization'] == 'Bearer new2' ? _json({'ok': true}) : _json({'code': 'unauthenticated', 'title': 'x'}, 401);
    });
    expect((await api.get('/v1/me/account'))['ok'], isTrue);
    expect(calls, 2, reason: 'one refresh at start-up (no access token yet) and one after the 401');
    expect(s.signedIn, isTrue);
    expect(reqs.where((r) => r.url.path == '/v1/auth/token/refresh').map((r) => jsonDecode(r.body)['refresh_token']), ['rt0', 'rt1'],
        reason: 'every refresh uses the token returned by the previous rotation');
  });

  test('a rejected refresh token (replay / revoked) signs the device out', () async {
    final (s, api, _) = await _setup((r) async =>
        r.url.path == '/v1/auth/token/refresh' ? _json({'code': 'unauthenticated', 'title': 'x'}, 401) : _json({'code': 'unauthenticated', 'title': 'x'}, 401));
    await expectLater(api.get('/v1/me/account'), throwsA(isA<ApiException>().having((e) => e.status, 'status', 401)));
    expect(s.signedIn, isFalse);
  });

  test('concurrent 401s share one refresh', () async {
    var refreshes = 0;
    final (_, api, _) = await _setup((r) async {
      if (r.url.path == '/v1/auth/token/refresh') {
        refreshes++;
        await Future<void>.delayed(const Duration(milliseconds: 10));
        return _json({'access_token': 'a', 'refresh_token': 'b', 'session_id': 's'});
      }
      return _json({'ok': true});
    });
    await Future.wait([api.get('/x'), api.get('/y'), api.get('/z')]);
    expect(refreshes, 1);
  });

  test('device id is stable per install', () async {
    final (s, _, _) = await _setup((r) async => _json({}), refresh: null);
    expect(s.deviceId, hasLength(32));
    expect(s.deviceId, s.deviceId);
    expect(s.signedIn, isFalse);
  });

  test('problem+json field errors are exposed', () {
    final e = ApiException.fromBody(422, {'code': 'validation_failed', 'title': 't', 'errors': [{'field': 'handle', 'code': 'taken'}]});
    expect(e.fieldErrors, {'handle': 'taken'});
  });

  test('post parsing: hidden like count stays null', () {
    final p = Post.fromJson({
      'id': 'p', 'author': {'id': 'a', 'handle': 'h', 'display_name': '', 'is_private': false}, 'caption': 'c', 'visibility': 'followers',
      'media': [{'id': 'm', 'url': '/v1/media/m', 'width': 1, 'height': 1}], 'like_count': null, 'comment_count': 2, 'viewer_liked': false,
      'is_owner': true, 'comments_allowed': true, 'published_at': '2026-10-28T10:00:00Z', 'version': 4,
    });
    expect(p.likeCount, isNull);
    expect(p.author.name, 'h');
    expect(p.copyWith(viewerLiked: true).likeCount, isNull);
  });
}
