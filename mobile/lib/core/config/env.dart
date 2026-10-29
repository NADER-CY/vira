/// Build-time configuration: `flutter run --dart-define=VIRA_API_BASE_URL=http://10.0.2.2:8080`.
/// 10.0.2.2 is the host machine from the Android emulator (the local compose stack).
abstract final class Env {
  static const apiBaseUrl = String.fromEnvironment('VIRA_API_BASE_URL', defaultValue: 'http://10.0.2.2:8080');
  static const appVersion = String.fromEnvironment('VIRA_APP_VERSION', defaultValue: '0.2.0');

  /// Version of the terms / privacy texts the user consents to (recorded server-side with each consent, UC-5).
  /// Placeholder until the legal texts are approved (doc 19 UC-5, owner decision D-13).
  static const consentTextVersion = String.fromEnvironment('VIRA_CONSENT_TEXT_VERSION', defaultValue: 'p2-draft-1');

  /// Minimum age confirmed at sign-up. D-13 (minimum age) is still OPEN: 18 is a placeholder, configurable per build.
  static const minimumAge = int.fromEnvironment('VIRA_MIN_AGE', defaultValue: 18);
}
