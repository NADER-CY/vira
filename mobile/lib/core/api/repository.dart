import 'dart:math';
import 'dart:typed_data';

import 'api_client.dart';
import 'models.dart';

/// Typed access to the Phase 2 public API (contracts/openapi/public.v1.json).
class Repo {
  Repo(this.api);
  final ApiClient api;

  static String newIdempotencyKey() {
    final r = Random.secure();
    return List.generate(16, (_) => r.nextInt(256).toRadixString(16).padLeft(2, '0')).join();
  }

  String _q(String path, Map<String, String?> q) {
    final params = {for (final e in q.entries) if (e.value != null && e.value!.isNotEmpty) e.key: e.value!};
    return params.isEmpty ? path : '$path?${Uri(queryParameters: params).query}';
  }

  // ── auth ────────────────────────────────────────────────────────────────
  Future<OtpChallenge> requestOtp(String phone, String deviceId) async {
    final res = await api.send('POST', '/v1/auth/otp',
        body: {'phone': phone, 'purpose': 'login'}, headers: {'X-Device-Id': deviceId}, authenticated: false);
    return OtpChallenge.fromJson(res.body);
  }

  /// Returns the SignIn body (tokens + user_id, handle, is_new).
  Future<Map<String, dynamic>> verifyOtp({
    required String challengeId,
    required String code,
    required Map<String, Object> device,
    required bool intendsToSell,
    required bool terms,
    required bool privacy,
    required bool ageConfirmation,
    required String textVersion,
  }) async {
    final res = await api.send('POST', '/v1/auth/otp/verify', authenticated: false, body: {
      'challenge_id': challengeId,
      'code': code,
      'device': device,
      'intends_to_sell': intendsToSell,
      'consents': {'terms': terms, 'privacy': privacy, 'age_confirmation': ageConfirmation, 'text_version': textVersion},
    });
    return res.body;
  }

  // ── account & sessions ──────────────────────────────────────────────────
  Future<Account> account() async => Account.fromJson(await api.get('/v1/me/account'));

  Future<List<DeviceSession>> sessions() async =>
      ApiPage.fromJson(await api.get('/v1/me/sessions'), DeviceSession.fromJson).items;

  Future<void> revokeSession(String id) => api.delete('/v1/me/sessions/$id');

  Future<void> revokeOtherSessions() => api.post('/v1/me/sessions/revoke-others', null);

  /// Schedules account deletion; returns when it takes effect (the grace period allows sign-in to cancel).
  Future<DateTime?> requestDeletion() async =>
      DateTime.tryParse((await api.post('/v1/me/deletion', {'confirm': true}))['effective_at'] as String? ?? '');

  // ── profile & privacy ───────────────────────────────────────────────────
  Future<Profile> myProfile() async => Profile.fromJson(await api.get('/v1/me/profile'));

  Future<Profile> updateProfile(int version, Map<String, Object?> changes) async =>
      Profile.fromJson(await api.patch('/v1/me/profile', changes, ifMatch: '"$version"'));

  Future<PrivacySetting> privacy() async => PrivacySetting.fromJson(await api.get('/v1/me/privacy'));

  Future<PrivacySetting> setPrivacy(PrivacySetting p) async => PrivacySetting.fromJson(await api.put('/v1/me/privacy', p.toJson()));

  Future<Profile> user(String id) async => Profile.fromJson(await api.get('/v1/users/$id'));

  Future<Profile> userByHandle(String handle) async => Profile.fromJson(await api.get('/v1/handles/${Uri.encodeComponent(handle)}'));

  Future<List<Author>> searchUsers(String q) async =>
      ApiPage.fromJson(await api.get(_q('/v1/users/search', {'q': q})), Author.fromJson).items;

  // ── graph ───────────────────────────────────────────────────────────────
  /// Returns 'following' or 'requested' (private account).
  Future<String> follow(String id) async => (await api.post('/v1/users/$id/follow', null))['state'] as String? ?? 'following';

  Future<void> unfollow(String id) => api.delete('/v1/users/$id/follow');

  Future<void> block(String id) => api.post('/v1/users/$id/block', null);

  Future<void> unblock(String id) => api.delete('/v1/users/$id/block');

  Future<void> removeFollower(String id) => api.delete('/v1/me/followers/$id');

  Future<ApiPage<ListEntry>> followers(String id, {String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q('/v1/users/$id/followers', {'cursor': cursor})), ListEntry.fromJson);

  Future<ApiPage<ListEntry>> following(String id, {String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q('/v1/users/$id/following', {'cursor': cursor})), ListEntry.fromJson);

  Future<ApiPage<ListEntry>> followRequests({String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q('/v1/me/follow-requests', {'cursor': cursor})), ListEntry.fromJson);

  Future<void> approveRequest(String id) => api.post('/v1/me/follow-requests/$id/approve', null);

  Future<void> declineRequest(String id) => api.post('/v1/me/follow-requests/$id/decline', null);

  Future<List<Author>> blocks() async => ApiPage.fromJson(await api.get('/v1/me/blocks'), Author.fromJson).items;

  // ── posts, feeds, engagement ────────────────────────────────────────────
  Future<ApiPage<Post>> feed({required bool following, String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q(following ? '/v1/feed/following' : '/v1/feed/home', {'cursor': cursor})), Post.fromJson);

  Future<ApiPage<Post>> userPosts(String id, {String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q('/v1/users/$id/posts', {'cursor': cursor})), Post.fromJson);

  Future<Post> post(String id) async => Post.fromJson(await api.get('/v1/posts/$id'));

  Future<Post> createPost({required String caption, required String visibility, required List<String> mediaIds, required String idempotencyKey}) async =>
      Post.fromJson(await api.post('/v1/posts', {'caption': caption, 'visibility': visibility, 'media_ids': mediaIds}, idempotencyKey: idempotencyKey));

  Future<Post> updatePost(Post p, {String? caption, String? visibility}) async => Post.fromJson(await api.patch('/v1/posts/${p.id}', {
        'caption': ?caption,
        'visibility': ?visibility,
      }, ifMatch: '"${p.version}"'));

  Future<void> deletePost(String id) => api.delete('/v1/posts/$id');

  /// Returns (liked, like_count — null when hidden).
  Future<(bool, int?)> setLike(String postId, bool like) async {
    final r = like ? await api.put('/v1/posts/$postId/like', null) : await api.delete('/v1/posts/$postId/like');
    return (r['liked'] == true, (r['like_count'] as num?)?.toInt());
  }

  Future<ApiPage<Comment>> comments(String postId, {String? cursor}) async =>
      ApiPage.fromJson(await api.get(_q('/v1/posts/$postId/comments', {'cursor': cursor})), Comment.fromJson);

  Future<Comment> addComment(String postId, String body, String idempotencyKey) async =>
      Comment.fromJson(await api.post('/v1/posts/$postId/comments', {'body': body}, idempotencyKey: idempotencyKey));

  Future<void> deleteComment(String id) => api.delete('/v1/comments/$id');

  // ── media & reports ─────────────────────────────────────────────────────
  Future<MediaInfo> uploadImage(Uint8List bytes, String filename, {required String purpose}) async =>
      MediaInfo.fromJson(await api.upload('/v1/media/images', bytes: bytes, filename: filename, fields: {'purpose': purpose}));

  Future<void> report({required String targetType, required String targetId, required String reason, String details = ''}) =>
      api.post('/v1/reports', {'target_type': targetType, 'target_id': targetId, 'reason': reason, if (details.isNotEmpty) 'details': details});
}
