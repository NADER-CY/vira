import 'dart:async';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';
import '../config/env.dart';

/// Where the refresh token lives. On devices: Android Keystore / iOS Keychain (flutter_secure_storage);
/// tests use [MemoryTokenStore].
abstract class TokenStore {
  Future<String?> read(String key);
  Future<void> write(String key, String? value);
}

class SecureTokenStore implements TokenStore {
  const SecureTokenStore();
  static const _s = FlutterSecureStorage();

  @override
  Future<String?> read(String key) => _s.read(key: key);

  @override
  Future<void> write(String key, String? value) async {
    if (value == null) {
      await _s.delete(key: key);
    } else {
      await _s.write(key: key, value: value);
    }
  }
}

class MemoryTokenStore implements TokenStore {
  MemoryTokenStore([Map<String, String>? initial]) : _m = {...?initial};
  final Map<String, String> _m;

  @override
  Future<String?> read(String key) async => _m[key];

  @override
  Future<void> write(String key, String? value) async {
    if (value == null) {
      _m.remove(key);
    } else {
      _m[key] = value;
    }
  }
}

/// Client side of the Phase 2 session model (doc 08 §2): short-lived access token in memory, rotating refresh
/// token in secure storage, one stable device id per install. Notifies listeners (the router) on sign-in/out.
class Session extends ChangeNotifier {
  Session({required this.api, required this.store, required this.prefs}) {
    api.auth = AuthHooks(accessToken: () => _access, refresh: refresh, deviceId: () => deviceId, canRefresh: () => _refresh != null);
  }

  static const _kRefresh = 'vira.refresh_token';
  static const _kUser = 'vira.user_id';
  static const _kHandle = 'vira.handle';
  static const _kSession = 'vira.session_id';
  static const _kDevice = 'vira.device_id';

  final ApiClient api;
  final TokenStore store;
  final SharedPreferences prefs;

  String? _access;
  String? _refresh;
  String? userId;
  String? handle;
  String? sessionId;
  Future<bool>? _refreshing;

  bool get signedIn => _refresh != null;

  /// A random identifier per installation (not a hardware id), sent as X-Device-Id and in the sign-in body.
  String get deviceId {
    var id = prefs.getString(_kDevice);
    if (id == null) {
      final r = Random.secure();
      id = List.generate(16, (_) => r.nextInt(256).toRadixString(16).padLeft(2, '0')).join();
      prefs.setString(_kDevice, id);
    }
    return id;
  }

  Map<String, Object> get deviceInfo => {
        'id': deviceId,
        'platform': defaultTargetPlatform == TargetPlatform.iOS ? 'ios' : 'android',
        'model': defaultTargetPlatform.name,
        'app_version': Env.appVersion,
      };

  Future<void> restore() async {
    _refresh = await store.read(_kRefresh);
    userId = prefs.getString(_kUser);
    handle = prefs.getString(_kHandle);
    sessionId = prefs.getString(_kSession);
    if (_refresh != null) notifyListeners();
  }

  /// Stores the result of POST /v1/auth/otp/verify (SignIn) or /v1/auth/token/refresh (Tokens).
  Future<void> accept(Map<String, dynamic> j) async {
    _access = j['access_token'] as String?;
    _refresh = j['refresh_token'] as String?;
    sessionId = j['session_id'] as String? ?? sessionId;
    userId = j['user_id'] as String? ?? userId;
    handle = j['handle'] as String? ?? handle;
    await store.write(_kRefresh, _refresh);
    await prefs.setString(_kSession, sessionId ?? '');
    if (userId != null) await prefs.setString(_kUser, userId!);
    if (handle != null) await prefs.setString(_kHandle, handle!);
    notifyListeners();
  }

  void updateHandle(String h) {
    handle = h;
    prefs.setString(_kHandle, h);
    notifyListeners();
  }

  /// Rotates the token pair once; concurrent callers share the same attempt. A rejected refresh token
  /// (expired, revoked, or replayed — the server then revokes the session) signs the device out.
  Future<bool> refresh() => _refreshing ??= _doRefresh().whenComplete(() => _refreshing = null);

  Future<bool> _doRefresh() async {
    final rt = _refresh;
    if (rt == null) return false;
    try {
      final res = await api.send('POST', '/v1/auth/token/refresh', body: {'refresh_token': rt}, authenticated: false);
      await accept(res.body);
      return true;
    } on ApiException catch (e) {
      if (e.status == 401 || e.status == 403) await _clear();
      return false;
    }
  }

  /// Signs out: revokes the session on the server (best effort) and forgets the tokens on the device.
  Future<void> signOut({bool remote = true}) async {
    if (remote && _refresh != null) {
      try {
        if (_access == null) await refresh();
        await api.post('/v1/auth/logout', null);
      } on ApiException {
        // The local sign-out still happens; the server session expires or is revoked from another device.
      }
    }
    await _clear();
  }

  Future<void> _clear() async {
    _access = null;
    _refresh = null;
    userId = null;
    handle = null;
    sessionId = null;
    await store.write(_kRefresh, null);
    await prefs.remove(_kUser);
    await prefs.remove(_kHandle);
    await prefs.remove(_kSession);
    notifyListeners();
  }
}
