import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:http/http.dart' as http;

/// API error parsed from an RFC 9457 problem+json response (doc 07 §1), or a client-side network failure.
class ApiException implements Exception {
  ApiException({required this.status, required this.code, required this.title, this.traceId, this.extra = const {}});

  factory ApiException.fromBody(int status, Map<String, dynamic> body) {
    final known = {'type', 'title', 'status', 'code', 'detail', 'trace_id', 'errors', 'instance'};
    return ApiException(
      status: status,
      code: body['code'] as String? ?? 'internal_error',
      title: body['title'] as String? ?? messageFor(body['code'] as String? ?? 'internal_error'),
      traceId: body['trace_id'] as String?,
      extra: {
        for (final e in body.entries)
          if (!known.contains(e.key)) e.key: e.value,
        if (body['errors'] != null) 'errors': body['errors'],
      },
    );
  }

  final int status;
  final String code;
  final String title;
  final String? traceId;
  final Map<String, dynamic> extra;

  /// Field errors of a 422 response: field → code.
  Map<String, String> get fieldErrors {
    final errs = extra['errors'];
    if (errs is! List) return const {};
    return {
      for (final e in errs)
        if (e is Map<String, dynamic>) '${e['field']}': '${e['code']}',
    };
  }

  @override
  String toString() => 'ApiException($status $code: $title)';
}

/// Persian fallbacks for error codes the client produces itself or cannot map.
String messageFor(String code) => switch (code) {
      'network_error' => 'اتصال به سرور برقرار نشد. اتصال اینترنت را بررسی کنید.',
      'timeout' => 'پاسخ سرور طول کشید. دوباره تلاش کنید.',
      'maintenance' => 'ویرا در حال به‌روزرسانی است.',
      'upgrade_required' => 'برای ادامه، اپلیکیشن را به‌روزرسانی کنید.',
      'feature_locked' => 'این قابلیت تا تأیید مراحل قانونی و امنیتی قفل است.',
      _ => 'خطای غیرمنتظره رخ داد. دوباره تلاش کنید.',
    };

/// Hooks the session layer installs: the current access token, and a refresh that rotates the token pair once
/// when the server answers 401 (returns false when the session is gone and the user must sign in again).
class AuthHooks {
  const AuthHooks({required this.accessToken, required this.refresh, required this.deviceId, required this.canRefresh});
  final String? Function() accessToken;
  final bool Function() canRefresh;
  final Future<bool> Function() refresh;
  final String Function() deviceId;
}

/// A JSON response body plus the headers the client needs (ETag for If-Match).
class ApiResponse {
  const ApiResponse(this.status, this.body, this.headers);
  final int status;
  final Map<String, dynamic> body;
  final Map<String, String> headers;
}

/// Thin JSON client over package:http: timeouts, Accept-Language fa-IR, bearer token with one transparent
/// refresh-and-retry on 401, idempotency keys and If-Match.
class ApiClient {
  ApiClient({required this.baseUrl, http.Client? client, this.timeout = const Duration(seconds: 15)})
      : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;
  final Duration timeout;
  AuthHooks? auth;

  Uri uri(String path) => Uri.parse('$baseUrl$path');

  /// Headers for authenticated binary GETs (images served by /v1/media/{id}).
  Map<String, String> get imageHeaders => {
        if (auth?.accessToken() case final t?) 'Authorization': 'Bearer $t',
      };

  Future<Map<String, dynamic>> get(String path) async => (await send('GET', path)).body;

  Future<Map<String, dynamic>> post(String path, Object? body, {String? idempotencyKey}) async =>
      (await send('POST', path, body: body, idempotencyKey: idempotencyKey)).body;

  Future<Map<String, dynamic>> put(String path, Object? body) async => (await send('PUT', path, body: body)).body;

  Future<Map<String, dynamic>> patch(String path, Object? body, {String? ifMatch}) async =>
      (await send('PATCH', path, body: body, ifMatch: ifMatch)).body;

  Future<Map<String, dynamic>> delete(String path) async => (await send('DELETE', path)).body;

  Future<ApiResponse> send(String method, String path,
      {Object? body, String? idempotencyKey, String? ifMatch, Map<String, String> headers = const {}, bool authenticated = true}) async {
    http.BaseRequest build() {
      final req = http.Request(method, uri(path))
        ..headers.addAll(_headers(json: body != null, extra: headers, bearer: authenticated));
      if (idempotencyKey != null) req.headers['Idempotency-Key'] = idempotencyKey;
      if (ifMatch != null) req.headers['If-Match'] = ifMatch;
      if (body != null) req.body = jsonEncode(body);
      return req;
    }

    return _withRefresh(build, retry: authenticated);
  }

  /// multipart/form-data upload (POST /v1/media/images).
  Future<Map<String, dynamic>> upload(String path, {required Uint8List bytes, required String filename, Map<String, String> fields = const {}}) async {
    http.BaseRequest build() => http.MultipartRequest('POST', uri(path))
      ..headers.addAll(_headers(json: false))
      ..fields.addAll(fields)
      ..files.add(http.MultipartFile.fromBytes('file', bytes, filename: filename));
    return (await _withRefresh(build, uploadTimeout: true)).body;
  }

  Map<String, String> _headers({required bool json, Map<String, String> extra = const {}, bool bearer = true}) => {
        'Accept': 'application/json',
        'Accept-Language': 'fa-IR',
        if (json) 'Content-Type': 'application/json',
        if (bearer)
          if (auth?.accessToken() case final t?) 'Authorization': 'Bearer $t',
        if (auth != null) 'X-Device-Id': auth!.deviceId(),
        ...extra,
      };

  Future<ApiResponse> _withRefresh(http.BaseRequest Function() build, {bool uploadTimeout = false, bool retry = true}) async {
    // After an app restart only the refresh token survives: obtain an access token before the first call.
    if (retry && auth != null && auth!.accessToken() == null && auth!.canRefresh()) await auth!.refresh();
    final hadToken = auth?.accessToken() != null;
    var res = await _once(build(), uploadTimeout);
    if (retry && res.status == 401 && hadToken && auth != null && await auth!.refresh()) {
      res = await _once(build(), uploadTimeout);
    }
    if (res.status >= 400) throw ApiException.fromBody(res.status, res.body);
    return res;
  }

  Future<ApiResponse> _once(http.BaseRequest req, bool long) async {
    http.Response res;
    try {
      res = await http.Response.fromStream(await _client.send(req).timeout(long ? timeout * 4 : timeout));
    } on TimeoutException {
      throw ApiException(status: 0, code: 'timeout', title: messageFor('timeout'));
    } on SocketException {
      throw ApiException(status: 0, code: 'network_error', title: messageFor('network_error'));
    } on http.ClientException {
      throw ApiException(status: 0, code: 'network_error', title: messageFor('network_error'));
    }
    final text = utf8.decode(res.bodyBytes);
    Object? decoded;
    try {
      decoded = text.isEmpty ? <String, dynamic>{} : jsonDecode(text);
    } on FormatException {
      decoded = <String, dynamic>{'code': 'internal_error'};
    }
    final map = decoded is Map<String, dynamic> ? decoded : <String, dynamic>{'data': decoded};
    return ApiResponse(res.statusCode, map, res.headers);
  }
}
