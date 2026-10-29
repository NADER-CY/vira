import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../report/report_screen.dart';

/// A post as drawn in the home feed (`_33` dark / `home_feed_light`): header (ringed avatar, handle, follow link,
/// more), 4:5 media with the "۱ / ۳" counter, action row (like, comment, share, bookmark), caption with hashtags,
/// comment link and time. Commerce overlays of the reference (product tags, buy-now card) arrive with Phase 4
/// data; they are not drawn because Phase 2 posts carry no products.
class PostCard extends ConsumerStatefulWidget {
  const PostCard({super.key, required this.post, this.onChanged, this.onDeleted, this.detail = false});
  final Post post;
  final ValueChanged<Post>? onChanged;
  final VoidCallback? onDeleted;
  final bool detail;

  @override
  ConsumerState<PostCard> createState() => _PostCardState();
}

class _PostCardState extends ConsumerState<PostCard> {
  late Post _p = widget.post;
  int _page = 0;
  bool _liking = false;

  @override
  void didUpdateWidget(PostCard old) {
    super.didUpdateWidget(old);
    if (old.post != widget.post) _p = widget.post;
  }

  void _set(Post p) {
    setState(() => _p = p);
    widget.onChanged?.call(p);
  }

  Future<void> _toggleLike() async {
    if (_liking) return;
    final want = !_p.viewerLiked;
    final before = _p;
    _liking = true;
    // Optimistic update; the server's answer is authoritative.
    _set(_p.copyWith(viewerLiked: want, likeCount: _p.likeCount == null ? null : _p.likeCount! + (want ? 1 : -1)));
    try {
      final (liked, count) = await ref.read(repoProvider).setLike(_p.id, want);
      _set(_p.copyWith(viewerLiked: liked, likeCount: count, clearLikeCount: count == null));
    } on ApiException catch (e) {
      _set(before);
      if (mounted) toast(context, e.title);
    } finally {
      _liking = false;
    }
  }

  Future<void> _follow() async {
    try {
      final state = await ref.read(repoProvider).follow(_p.author.id);
      if (mounted) toast(context, state == 'requested' ? 'درخواست دنبال کردن ارسال شد.' : 'دنبال می‌کنید.');
      _set(Post.fromJson({..._toJson(_p), 'viewer_follows_author': state == 'following'}));
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    }
  }

  Future<void> _menu() async {
    final items = <(IconData, String, String, bool)>[
      if (_p.isOwner) (Sym.edit, 'ویرایش پست', 'edit', false),
      if (_p.isOwner) (Sym.delete, 'حذف پست', 'delete', true),
      if (!_p.isOwner) (Sym.person, 'مشاهده نمایه', 'profile', false),
      if (!_p.isOwner) (Sym.flag, 'گزارش پست', 'report', true),
    ];
    final choice = await showActionSheet<String>(context, items);
    if (!mounted || choice == null) return;
    switch (choice) {
      case 'edit':
        final updated = await context.push<Post>('/post/${_p.id}/edit', extra: _p);
        if (updated != null) _set(updated);
      case 'delete':
        if (!await confirm(context, title: 'حذف پست', message: 'این پست برای همیشه حذف می‌شود.', action: 'حذف', danger: true)) return;
        try {
          await ref.read(repoProvider).deletePost(_p.id);
          widget.onDeleted?.call();
          if (mounted) toast(context, 'پست حذف شد.');
        } on ApiException catch (e) {
          if (mounted) toast(context, e.title);
        }
      case 'profile':
        context.push('/u/${_p.author.id}');
      case 'report':
        showReportSheet(context, targetType: 'post', targetId: _p.id);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final sk = context.skin;
    final p = _p;
    return Container(
      decoration: BoxDecoration(
        color: sk.card,
        borderRadius: BorderRadius.circular(16),
        border: sk.cardBorder == null ? null : Border.all(color: sk.cardBorder!),
        boxShadow: sk.cardShadow,
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        // header — p-3.5, avatar 40 with gradient ring
        Padding(
          padding: const EdgeInsets.all(14),
          child: Row(children: [
            GestureDetector(
              onTap: () => context.push('/u/${p.author.id}'),
              child: Avatar(author: p.author, size: 40, ring: storyRing(cs)),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, mainAxisSize: MainAxisSize.min, children: [
                Row(children: [
                  Flexible(
                    child: GestureDetector(
                      onTap: () => context.push('/u/${p.author.id}'),
                      child: Text(isolate(p.author.handle), style: tw(14, 20, 600, sk.text), maxLines: 1, overflow: TextOverflow.ellipsis),
                    ),
                  ),
                  if (!p.isOwner && !p.viewerFollowsAuthor) ...[
                    const SizedBox(width: 8),
                    GestureDetector(
                      key: Key('follow-${p.id}'),
                      onTap: _follow,
                      child: Text('• دنبال کردن', style: tw(11, 14, 500, sk.muted)),
                    ),
                  ],
                ]),
                if (p.author.displayName.isNotEmpty)
                  Text(p.author.displayName, style: tw(11, 14, 500, sk.muted), maxLines: 1, overflow: TextOverflow.ellipsis),
              ]),
            ),
            RoundIconButton(icon: Sym.moreHoriz, label: 'گزینه‌ها', size: 36, iconSize: 20, color: sk.muted, onTap: _menu),
          ]),
        ),
        if (p.media.isNotEmpty) _media(cs, sk, p),
        // actions — px-3.5 pt-3 pb-2, gap-4
        Padding(
          padding: const EdgeInsets.fromLTRB(14, 12, 14, 8),
          child: Row(children: [
            _Action(
              key: Key('like-${p.id}'),
              icon: Sym.favorite,
              size: 26,
              fill: p.viewerLiked,
              color: p.viewerLiked ? sk.accent : sk.text,
              label: p.likeCount == null ? null : formatNumber(p.likeCount!),
              semantic: p.viewerLiked ? 'برداشتن پسند' : 'پسندیدن',
              textColor: sk.text,
              onTap: _toggleLike,
            ),
            const SizedBox(width: 16),
            _Action(
              icon: Sym.chatBubble,
              color: sk.text,
              textColor: sk.text,
              label: formatNumber(p.commentCount),
              semantic: 'نظرها',
              onTap: () => context.push('/post/${p.id}/comments', extra: p),
            ),
            const SizedBox(width: 16),
            _Action(icon: Sym.share, color: sk.text, textColor: sk.text, semantic: 'اشتراک‌گذاری', onTap: () => soon(context, 'اشتراک‌گذاری پست', 'فاز ۳')),
            const Spacer(),
            _Action(icon: Sym.bookmark, color: sk.text, textColor: sk.text, semantic: 'ذخیره', onTap: () => soon(context, 'ذخیرهٔ پست', 'فاز ۳')),
          ]),
        ),
        if (p.caption.isNotEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 0, 14, 8),
            child: _Caption(handle: p.author.handle, caption: p.caption, text: sk.text, accent: sk.accent, full: widget.detail),
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(14, 0, 14, 10),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            if (p.commentCount > 0 && !widget.detail)
              GestureDetector(
                onTap: () => context.push('/post/${p.id}/comments', extra: p),
                child: Padding(
                  padding: const EdgeInsets.only(top: 2),
                  child: Text('مشاهده تمام ${formatNumber(p.commentCount)} نظر...', style: tw(11, 14, 500, sk.muted)),
                ),
              ),
            if (!p.commentsAllowed)
              Padding(
                padding: const EdgeInsets.only(top: 2),
                child: Text('نظرها برای این پست بسته است.', style: tw(11, 14, 500, sk.muted)),
              ),
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                '${formatRelative(p.publishedAt)}${p.editedAt != null ? ' · ویرایش‌شده' : ''}${p.visibility == 'followers' ? ' · فقط دنبال‌کنندگان' : ''}',
                style: tw(11, 20, 400, sk.muted.withValues(alpha: 0.7)),
              ),
            ),
          ]),
        ),
      ]),
    );
  }

  Widget _media(ColorScheme cs, FeedSkin sk, Post p) => AspectRatio(
        aspectRatio: 4 / 5,
        child: Stack(children: [
          Positioned.fill(
            child: ColoredBox(
              color: sk.media,
              child: PageView.builder(
                itemCount: p.media.length,
                onPageChanged: (i) => setState(() => _page = i),
                itemBuilder: (_, i) => GestureDetector(
                  onDoubleTap: p.viewerLiked ? null : _toggleLike,
                  child: NetImage(p.media[i].url, placeholder: sk.media),
                ),
              ),
            ),
          ),
          if (p.media.length > 1)
            Positioned(
              top: 14,
              left: 14,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest.withValues(alpha: 0.8),
                  borderRadius: BorderRadius.circular(999),
                  boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
                ),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Sy(Sym.photoLibrary, size: 14, color: cs.onSurface),
                  const SizedBox(width: 4),
                  Text('${toPersianDigits(_page + 1)} / ${toPersianDigits(p.media.length)}', style: tw(11, 14, 500, cs.onSurface)),
                ]),
              ),
            ),
        ]),
      );

  static Map<String, dynamic> _toJson(Post p) => {
        'id': p.id,
        'author': {'id': p.author.id, 'handle': p.author.handle, 'display_name': p.author.displayName, 'avatar_url': p.author.avatarUrl, 'is_private': p.author.isPrivate},
        'caption': p.caption,
        'visibility': p.visibility,
        'media': [for (final m in p.media) {'id': m.id, 'url': m.url, 'width': m.width, 'height': m.height}],
        'like_count': p.likeCount,
        'comment_count': p.commentCount,
        'viewer_liked': p.viewerLiked,
        'is_owner': p.isOwner,
        'comments_allowed': p.commentsAllowed,
        'published_at': p.publishedAt.toIso8601String(),
        'edited_at': p.editedAt?.toIso8601String(),
        'version': p.version,
        'viewer_follows_author': p.viewerFollowsAuthor,
      };
}

class _Action extends StatelessWidget {
  const _Action({super.key, required this.icon, required this.color, required this.textColor, required this.semantic, required this.onTap, this.size = 24, this.fill = false, this.label});
  final IconData icon;
  final Color color, textColor;
  final String semantic;
  final VoidCallback onTap;
  final double size;
  final bool fill;
  final String? label;

  @override
  Widget build(BuildContext context) => Semantics(
        button: true,
        label: semantic,
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: onTap,
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            ExcludeSemantics(child: Sy(icon, size: size, color: color, fill: fill)),
            if (label != null) ...[const SizedBox(width: 6), Text(label!, style: tw(12, 16, 600, textColor))],
          ]),
        ),
      );
}

/// Caption: bold handle, text, then the hashtags as a separate wrapped row in the accent colour.
class _Caption extends StatelessWidget {
  const _Caption({required this.handle, required this.caption, required this.text, required this.accent, required this.full});
  final String handle, caption;
  final Color text, accent;
  final bool full;

  @override
  Widget build(BuildContext context) {
    final tags = RegExp(r'#[^\s#]+').allMatches(caption).map((m) => m.group(0)!).toList();
    final body = caption.replaceAll(RegExp(r'#[^\s#]+'), '').trim();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Text.rich(
        TextSpan(children: [
          TextSpan(text: '${isolate(handle)} ', style: tw(14, 20, 700, text)),
          TextSpan(text: body, style: tw(14, 22.75, 400, text)),
        ]),
        maxLines: full ? null : 3,
        overflow: full ? TextOverflow.visible : TextOverflow.ellipsis,
      ),
      if (tags.isNotEmpty)
        Padding(
          padding: const EdgeInsets.only(top: 6),
          child: Wrap(spacing: 6, runSpacing: 2, children: [for (final t in tags) Text(t, style: tw(11, 14, 500, accent))]),
        ),
    ]);
  }
}
