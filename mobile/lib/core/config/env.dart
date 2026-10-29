/// Build-time configuration: `flutter run --dart-define=VIRA_API_BASE_URL=http://10.0.2.2:8080`.
/// 10.0.2.2 is the host machine from the Android emulator (the local compose stack).
abstract final class Env {
  static const apiBaseUrl = String.fromEnvironment('VIRA_API_BASE_URL', defaultValue: 'http://10.0.2.2:8080');
  static const appVersion = String.fromEnvironment('VIRA_APP_VERSION', defaultValue: '0.1.0');
}
