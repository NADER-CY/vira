import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'api/api_client.dart';
import 'api/repository.dart';
import 'auth/session.dart';
import 'config/env.dart';

/// Overridden in main() with the loaded instance; tests override it with mock values.
final sharedPreferencesProvider = Provider<SharedPreferences>((ref) => throw UnimplementedError('override in main'));

final apiClientProvider = Provider<ApiClient>((ref) => ApiClient(baseUrl: Env.apiBaseUrl));

/// Refresh-token storage: Keystore/Keychain on devices, overridden with [MemoryTokenStore] in tests.
final tokenStoreProvider = Provider<TokenStore>((ref) => const SecureTokenStore());

/// The signed-in session (a ChangeNotifier the router listens to). Restored in main() before the first frame.
final sessionProvider = Provider<Session>((ref) {
  final s = Session(api: ref.watch(apiClientProvider), store: ref.watch(tokenStoreProvider), prefs: ref.watch(sharedPreferencesProvider));
  ref.onDispose(s.dispose);
  return s;
});

final repoProvider = Provider<Repo>((ref) => Repo(ref.watch(apiClientProvider)));

/// Light / dark / system theme, persisted on the device (doc 09: both themes are required).
class ThemeModeNotifier extends Notifier<ThemeMode> {
  static const _key = 'vira.theme';

  @override
  ThemeMode build() => switch (ref.read(sharedPreferencesProvider).getString(_key)) {
        'light' => ThemeMode.light,
        'dark' => ThemeMode.dark,
        _ => ThemeMode.system,
      };

  Future<void> set(ThemeMode mode) async {
    state = mode;
    await ref.read(sharedPreferencesProvider).setString(_key, mode.name);
  }
}

final themeModeProvider = NotifierProvider<ThemeModeNotifier, ThemeMode>(ThemeModeNotifier.new);

/// Client configuration from GET /v1/config: non-regulated flags and the regulated capabilities that are
/// locked until their gate. The app hides or labels locked features; it never enables them locally.
class ClientConfig {
  const ClientConfig({required this.flags, required this.regulatedLocked});

  factory ClientConfig.fromJson(Map<String, dynamic> j) => ClientConfig(
        flags: (j['flags'] as Map<String, dynamic>? ?? const {}).map((k, v) => MapEntry(k, v == true)),
        regulatedLocked: (j['regulated_locked'] as Map<String, dynamic>? ?? const {}).map((k, v) => MapEntry(k, '$v')),
      );

  /// Used when the server cannot be reached: every regulated capability is treated as locked.
  static const offline = ClientConfig(flags: {}, regulatedLocked: {});

  final Map<String, bool> flags;
  final Map<String, String> regulatedLocked;

  bool isOn(String flag) => flags[flag] ?? false;
}

final clientConfigProvider = FutureProvider<ClientConfig>((ref) async {
  final json = await ref.read(apiClientProvider).get('/v1/config');
  return ClientConfig.fromJson(json);
});
