// Phase 2 API models (contracts/openapi/public.v1.json). Parsing is lenient: missing optional fields default.

String _s(Object? v) => v is String ? v : '';
int _i(Object? v) => v is num ? v.toInt() : 0;
bool _b(Object? v) => v == true;
DateTime? _t(Object? v) => v is String ? DateTime.tryParse(v) : null;
Map<String, dynamic> _m(Object? v) => v is Map<String, dynamic> ? v : const {};
List<Map<String, dynamic>> _l(Object? v) => v is List ? [for (final e in v) if (e is Map<String, dynamic>) e] : const [];

class Author {
  const Author({required this.id, required this.handle, required this.displayName, this.avatarUrl, this.isPrivate = false});

  factory Author.fromJson(Map<String, dynamic> j) => Author(
        id: _s(j['id']),
        handle: _s(j['handle']),
        displayName: _s(j['display_name']),
        avatarUrl: j['avatar_url'] as String?,
        isPrivate: _b(j['is_private']),
      );

  final String id, handle, displayName;
  final String? avatarUrl;
  final bool isPrivate;

  String get name => displayName.isEmpty ? handle : displayName;
}

class Stats {
  const Stats({this.posts = 0, this.followers = 0, this.following = 0});
  factory Stats.fromJson(Map<String, dynamic> j) => Stats(posts: _i(j['posts']), followers: _i(j['followers']), following: _i(j['following']));
  final int posts, followers, following;
}

class Relationship {
  const Relationship({this.isSelf = false, this.following = false, this.requested = false, this.followsYou = false, this.blocking = false});
  factory Relationship.fromJson(Map<String, dynamic> j) => Relationship(
        isSelf: _b(j['is_self']),
        following: _b(j['following']),
        requested: _b(j['requested']),
        followsYou: _b(j['follows_you']),
        blocking: _b(j['blocking']),
      );
  final bool isSelf, following, requested, followsYou, blocking;

  Relationship copyWith({bool? following, bool? requested, bool? blocking}) => Relationship(
        isSelf: isSelf,
        following: following ?? this.following,
        requested: requested ?? this.requested,
        followsYou: followsYou,
        blocking: blocking ?? this.blocking,
      );
}

class PrivacySetting {
  const PrivacySetting({this.isPrivate = false, this.commentsFrom = 'everyone', this.hideLikeCounts = false, this.settings = const {}});
  factory PrivacySetting.fromJson(Map<String, dynamic> j) => PrivacySetting(
        isPrivate: _b(j['is_private']),
        commentsFrom: j['comments_from'] as String? ?? 'everyone',
        hideLikeCounts: _b(j['hide_like_counts']),
        settings: _m(j['settings']),
      );
  final bool isPrivate, hideLikeCounts;
  final String commentsFrom;
  final Map<String, dynamic> settings;

  Map<String, dynamic> toJson() => {'is_private': isPrivate, 'comments_from': commentsFrom, 'hide_like_counts': hideLikeCounts, 'settings': settings};

  PrivacySetting copyWith({bool? isPrivate, String? commentsFrom, bool? hideLikeCounts}) => PrivacySetting(
        isPrivate: isPrivate ?? this.isPrivate,
        commentsFrom: commentsFrom ?? this.commentsFrom,
        hideLikeCounts: hideLikeCounts ?? this.hideLikeCounts,
        settings: settings,
      );
}

class Profile {
  const Profile({
    required this.author,
    this.bio = '',
    this.website = '',
    this.stats = const Stats(),
    this.relationship = const Relationship(),
    this.canViewPosts = true,
    this.version = 0,
    this.privacy,
  });

  factory Profile.fromJson(Map<String, dynamic> j) => Profile(
        author: Author.fromJson(j),
        bio: _s(j['bio']),
        website: _s(j['website']),
        stats: Stats.fromJson(_m(j['stats'])),
        relationship: Relationship.fromJson(_m(j['relationship'])),
        canViewPosts: j['can_view_posts'] != false,
        version: _i(j['version']),
        privacy: j['privacy'] is Map<String, dynamic> ? PrivacySetting.fromJson(j['privacy'] as Map<String, dynamic>) : null,
      );

  final Author author;
  final String bio, website;
  final Stats stats;
  final Relationship relationship;
  final bool canViewPosts;
  final int version;
  final PrivacySetting? privacy;

  String get id => author.id;
}

class MediaInfo {
  const MediaInfo({required this.id, required this.url, this.width = 0, this.height = 0});
  factory MediaInfo.fromJson(Map<String, dynamic> j) => MediaInfo(id: _s(j['id']), url: _s(j['url']), width: _i(j['width']), height: _i(j['height']));
  final String id, url;
  final int width, height;
}

class Post {
  const Post({
    required this.id,
    required this.author,
    required this.caption,
    required this.visibility,
    required this.media,
    required this.likeCount,
    required this.commentCount,
    required this.viewerLiked,
    required this.isOwner,
    required this.commentsAllowed,
    required this.publishedAt,
    required this.version,
    this.editedAt,
    this.viewerFollowsAuthor = false,
  });

  factory Post.fromJson(Map<String, dynamic> j) => Post(
        id: _s(j['id']),
        author: Author.fromJson(_m(j['author'])),
        caption: _s(j['caption']),
        visibility: j['visibility'] as String? ?? 'public',
        media: [for (final m in _l(j['media'])) MediaInfo.fromJson(m)],
        likeCount: j['like_count'] is num ? (j['like_count'] as num).toInt() : null,
        commentCount: _i(j['comment_count']),
        viewerLiked: _b(j['viewer_liked']),
        isOwner: _b(j['is_owner']),
        commentsAllowed: j['comments_allowed'] != false,
        publishedAt: _t(j['published_at']) ?? DateTime.fromMillisecondsSinceEpoch(0),
        editedAt: _t(j['edited_at']),
        version: _i(j['version']),
        viewerFollowsAuthor: _b(j['viewer_follows_author']),
      );

  final String id, caption, visibility;
  final Author author;
  final List<MediaInfo> media;

  /// null when the author hides like counts.
  final int? likeCount;
  final int commentCount, version;
  final bool viewerLiked, isOwner, commentsAllowed, viewerFollowsAuthor;
  final DateTime publishedAt;
  final DateTime? editedAt;

  Post copyWith({bool? viewerLiked, int? likeCount, bool clearLikeCount = false, int? commentCount, String? caption, String? visibility, int? version}) => Post(
        id: id,
        author: author,
        caption: caption ?? this.caption,
        visibility: visibility ?? this.visibility,
        media: media,
        likeCount: clearLikeCount ? null : (likeCount ?? this.likeCount),
        commentCount: commentCount ?? this.commentCount,
        viewerLiked: viewerLiked ?? this.viewerLiked,
        isOwner: isOwner,
        commentsAllowed: commentsAllowed,
        publishedAt: publishedAt,
        editedAt: editedAt,
        version: version ?? this.version,
        viewerFollowsAuthor: viewerFollowsAuthor,
      );
}

class Comment {
  const Comment({required this.id, required this.postId, required this.author, required this.body, required this.createdAt, this.canDelete = false, this.isPostAuthor = false});
  factory Comment.fromJson(Map<String, dynamic> j) => Comment(
        id: _s(j['id']),
        postId: _s(j['post_id']),
        author: Author.fromJson(_m(j['author'])),
        body: _s(j['body']),
        createdAt: _t(j['created_at']) ?? DateTime.fromMillisecondsSinceEpoch(0),
        canDelete: _b(j['can_delete']),
        isPostAuthor: _b(j['is_post_author']),
      );
  final String id, postId, body;
  final Author author;
  final DateTime createdAt;
  final bool canDelete, isPostAuthor;
}

class ListEntry {
  const ListEntry({required this.author, required this.following, this.since});
  factory ListEntry.fromJson(Map<String, dynamic> j) => ListEntry(author: Author.fromJson(j), following: _b(j['following']), since: _t(j['since']));
  final Author author;
  final bool following;
  final DateTime? since;
}

class Account {
  const Account({required this.userId, required this.status, required this.phoneMasked, this.intendsToSell = false, this.createdAt, this.deletionEffectiveAt});
  factory Account.fromJson(Map<String, dynamic> j) => Account(
        userId: _s(j['user_id']),
        status: j['status'] as String? ?? 'active',
        phoneMasked: _s(j['phone_masked']),
        intendsToSell: _b(j['intends_to_sell']),
        createdAt: _t(j['created_at']),
        deletionEffectiveAt: _t(j['deletion_effective_at']),
      );
  final String userId, status, phoneMasked;
  final bool intendsToSell;
  final DateTime? createdAt, deletionEffectiveAt;
}

class DeviceSession {
  const DeviceSession({required this.id, required this.platform, this.model = '', this.appVersion = '', this.createdAt, this.lastUsedAt, this.current = false});
  factory DeviceSession.fromJson(Map<String, dynamic> j) => DeviceSession(
        id: _s(j['id']),
        platform: _s(j['platform']),
        model: _s(j['model']),
        appVersion: _s(j['app_version']),
        createdAt: _t(j['created_at']),
        lastUsedAt: _t(j['last_used_at']),
        current: _b(j['current']),
      );
  final String id, platform, model, appVersion;
  final DateTime? createdAt, lastUsedAt;
  final bool current;
}

/// A page of a keyset-paginated list (`{data, next_cursor}`).
class ApiPage<T> {
  const ApiPage(this.items, this.next);
  factory ApiPage.fromJson(Map<String, dynamic> j, T Function(Map<String, dynamic>) item) =>
      ApiPage([for (final e in _l(j['data'])) item(e)], j['next_cursor'] as String?);
  final List<T> items;
  final String? next;
}

class OtpChallenge {
  const OtpChallenge({required this.id, required this.expiresIn, required this.resendAfter, required this.codeLength});
  factory OtpChallenge.fromJson(Map<String, dynamic> j) => OtpChallenge(
        id: _s(j['challenge_id']),
        expiresIn: _i(j['expires_in']),
        resendAfter: _i(j['resend_after']),
        codeLength: j['code_length'] is num ? _i(j['code_length']) : 6,
      );
  final String id;
  final int expiresIn, resendAfter, codeLength;
}
