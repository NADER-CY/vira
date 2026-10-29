import 'dart:async';
import 'dart:convert';
import 'dart:io';

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
      extra: {for (final e in body.entries) if (!known.contains(e.key)) e.key: e.value},
    );
  }

  final int status;
  final String code;
  final String title;
  final String? traceId;
  final Map<String, dynamic> extra;

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

/// Thin JSON client over package:http with timeouts, Accept-Language fa-IR, bearer token and idempotency keys.
class ApiClient {
  ApiClient({required this.baseUrl, http.Client? client, this.token, this.timeout = const Duration(seconds: 15)})
      : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;
  final String? Function()? token;
  final Duration timeout;

  Future<Map<String, dynamic>> get(String path) => _send('GET', path);

  Future<Map<String, dynamic>> post(String path, Object? body, {String? idempotencyKey}) =>
      _send('POST', path, body: body, idempotencyKey: idempotencyKey);

  Future<Map<String, dynamic>> _send(String method, String path, {Object? body, String? idempotencyKey}) async {
    final req = http.Request(method, Uri.parse('$baseUrl$path'))
      ..headers.addAll({
        'Accept': 'application/json',
        'Accept-Language': 'fa-IR',
        if (body != null) 'Content-Type': 'application/json',
        'Idempotency-Key': ?idempotencyKey,
        if (token?.call() case final t?) 'Authorization': 'Bearer $t',
      });
    if (body != null) req.body = jsonEncode(body);
    http.Response res;
    try {
      res = await http.Response.fromStream(await _client.send(req).timeout(timeout));
    } on TimeoutException {
      throw ApiException(status: 0, code: 'timeout', title: messageFor('timeout'));
    } on SocketException {
      throw ApiException(status: 0, code: 'network_error', title: messageFor('network_error'));
    } on http.ClientException {
      throw ApiException(status: 0, code: 'network_error', title: messageFor('network_error'));
    }
    final text = utf8.decode(res.bodyBytes);
    final decoded = text.isEmpty ? <String, dynamic>{} : jsonDecode(text);
    final map = decoded is Map<String, dynamic> ? decoded : <String, dynamic>{'data': decoded};
    if (res.statusCode >= 400) throw ApiException.fromBody(res.statusCode, map);
    return map;
  }
}
