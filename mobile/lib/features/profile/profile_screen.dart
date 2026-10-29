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
import '../../core/ui/paged.dart';
import '../../core/widgets/states.dart';
import '../home/home_screen.dart';
import '../report/report_screen.dart';

/// User profile. Reference: `user_profile_light` (M-276): header, avatar with gradient ring, stats, name, @handle,
/// bio, link, action buttons, content tabs (posts · reels · shop · tagged) and the 3-column grid. The reference
/// shows a store; store-only blocks (rating, official badge, top-seller card, storefront button, product chips)
/// come with Phase 4 seller data and are not drawn for Phase 2 accounts. No dark reference exists: the dark
/// variant maps the same inline colours to the dark home skin (OD-P2-04).
class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key, this.userId});

  /// null = the signed-in user (profile tab).
  final String? userId;

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  Profile? _profile;
  Object? _error;
  Paged<Post>? _posts;
  bool _busy = false;

  bool get _isMe => widget.userId == null || widget.userId == ref.read(sessionProvider).userId;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _posts?.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final repo = ref.read(repoProvider);
      final p = _isMe ? await repo.myProfile() : await repo.user(widget.userId!);
      if (!mounted) return;
      _posts?.dispose();
      _posts = p.canViewPosts ? (Paged<Post>((c) => repo.userPosts(p.id, cursor: c))..more()) : null;
      setState(() => _profile = p);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  Future<void> _followToggle() async {
    final p = _profile!;
    final r = p.relationship;
    setState(() => _busy = true);
    try {
      final repo = ref.read(repoProvider);
      if (r.following || r.requested) {
        if (r.following && p.author.isPrivate && !await confirm(context, title: 'لغو دنبال کردن', message: 'این حساب خصوصی است؛ برای دیدن دوباره پست‌ها باید درخواست بدهید.', action: 'لغو دنبال کردن')) {
          return;
        }
        await repo.unfollow(p.id);
      } else {
        await repo.follow(p.id);
      }
      await _load();
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _more() async {
    final p = _profile!;
    final blocking = p.relationship.blocking;
    final choice = await showActionSheet<String>(context, [
      if (p.relationship.followsYou) (Sym.personRemove, 'حذف از دنبال‌کنندگان', 'remove', false),
      (Sym.block, blocking ? 'رفع مسدودی' : 'مسدود کردن', 'block', !blocking),
      (Sym.flag, 'گزارش حساب', 'report', true),
    ]);
    if (!mounted || choice == null) return;
    final repo = ref.read(repoProvider);
    try {
      switch (choice) {
        case 'remove':
          await repo.removeFollower(p.id);
          if (mounted) toast(context, 'از دنبال‌کنندگان شما حذف شد.');
        case 'block':
          if (!blocking &&
              !await confirm(context, title: 'مسدود کردن ${p.author.handle}', message: 'این حساب دیگر نمی‌تواند نمایه و پست‌های شما را ببیند یا با شما تعامل کند. دنبال کردن‌های دوطرفه حذف می‌شود.', action: 'مسدود کردن', danger: true)) {
            return;
          }
          blocking ? await repo.unblock(p.id) : await repo.block(p.id);
          if (mounted) toast(context, blocking ? 'مسدودی برداشته شد.' : 'حساب مسدود شد.');
        case 'report':
          if (mounted) showReportSheet(context, targetType: 'user', targetId: p.id);
          return;
      }
      await _load();
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    }
  }

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    final p = _profile;
    return Scaffold(
      backgroundColor: sk.page,
      appBar: widget.userId == null ? const FeedHeader(title: 'نمایه') : ArchiveHeader(title: p == null ? '' : isolate('@${p.author.handle}'), horizontal: 12),
      body: p == null
          ? (_error == null ? const LoadingView(rows: 4) : Center(child: ErrorView(error: _error!, onRetry: _load)))
          : RefreshIndicator(
              color: sk.accent,
              onRefresh: _load,
              child: _body(context, p),
            ),
    );
  }

  Widget _body(BuildContext context, Profile p) {
    final cs = context.cs;
    final sk = context.skin;
    final posts = _posts;
    return CustomScrollView(slivers: [
      SliverToBoxAdapter(
        child: Container(
          decoration: BoxDecoration(gradient: LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter, colors: [sk.panel, sk.panel.withValues(alpha: 0)])),
          padding: const EdgeInsets.fromLTRB(14, 12, 14, 8),
          child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Row(children: [
              Container(
                width: 80,
                height: 80,
                padding: const EdgeInsets.all(3),
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: const LinearGradient(begin: Alignment.bottomLeft, end: Alignment.topRight, colors: [Color(0xFFFF2D65), Color(0xFF7B61FF), Color(0xFFFF2D65)]),
                  boxShadow: [BoxShadow(color: const Color(0xFFFF2D65).withValues(alpha: 0.15), blurRadius: 6, offset: const Offset(0, 4), spreadRadius: -1)],
                ),
                child: Avatar(author: p.author, size: 74, gap: 2, gapColor: sk.page),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                  _Stat(value: formatNumber(p.stats.posts), label: 'پست‌ها'),
                  _Stat(key: const Key('stat-followers'), value: formatNumber(p.stats.followers), label: 'دنبال‌کننده', onTap: p.canViewPosts ? () => context.push('/u/${p.id}/followers?tab=followers') : null),
                  _Stat(key: const Key('stat-following'), value: formatNumber(p.stats.following), label: 'دنبال‌شونده', onTap: p.canViewPosts ? () => context.push('/u/${p.id}/followers?tab=following') : null),
                ]),
              ),
            ]),
            const SizedBox(height: 12),
            Row(children: [
              Flexible(child: Text(p.author.name, style: tw(18, 27, 700, sk.text), overflow: TextOverflow.ellipsis)),
              if (p.author.isPrivate) ...[const SizedBox(width: 6), Sy(Sym.lock, size: 16, color: sk.muted)],
            ]),
            Text(isolate('@${p.author.handle}'), style: tw(13, 19.5, 600, sk.accent), textAlign: TextAlign.start),
            if (p.bio.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(p.bio, style: tw(13, 21.125, 400, sk.text)),
            ],
            if (p.website.isNotEmpty) ...[
              const SizedBox(height: 6),
              Row(children: [
                Sy(Sym.link, size: 15, color: sk.accent),
                const SizedBox(width: 6),
                Flexible(child: Text(isolate(p.website), style: tw(12, 18, 500, sk.accent), overflow: TextOverflow.ellipsis)),
              ]),
            ],
            if (p.relationship.followsYou && !_isMe) ...[
              const SizedBox(height: 6),
              Text('شما را دنبال می‌کند', style: tw(11, 16.5, 500, sk.muted)),
            ],
            const SizedBox(height: 14),
            _actions(context, p),
          ]),
        ),
      ),
      // content tabs
      SliverToBoxAdapter(
        child: Container(
          height: 46,
          decoration: BoxDecoration(color: sk.page, border: Border.symmetric(horizontal: BorderSide(color: sk.cardBorder ?? cs.outlineVariant.withValues(alpha: 0.4)))),
          child: Row(children: [
            _Tab(icon: Sym.gridOn, active: true, onTap: () {}),
            _Tab(icon: Sym.movie, active: false, onTap: () => soon(context, 'ریلزهای نمایه', 'فاز ۳')),
            _Tab(icon: Sym.shoppingBag, active: false, onTap: () => soon(context, 'فروشگاه نمایه', 'فاز ۴')),
            _Tab(icon: Sym.assignmentInd, active: false, onTap: () => soon(context, 'پست‌های برچسب‌خورده', 'فاز ۳')),
          ]),
        ),
      ),
      if (p.relationship.blocking)
        const SliverToBoxAdapter(child: EmptyView(icon: Sym.block, title: 'این حساب را مسدود کرده‌اید', message: 'برای دیدن پست‌ها، مسدودی را از منوی نمایه بردارید.'))
      else if (!p.canViewPosts)
        const SliverToBoxAdapter(child: EmptyView(icon: Sym.lock, title: 'این حساب خصوصی است', message: 'برای دیدن پست‌ها و دنبال‌کنندگان، درخواست دنبال کردن بفرستید.'))
      else if (posts != null)
        ListenableBuilder(
          listenable: posts,
          builder: (context, _) {
            if (!posts.loadedOnce) return const SliverToBoxAdapter(child: LoadingView(rows: 2));
            if (posts.error != null && posts.items.isEmpty) return SliverToBoxAdapter(child: ErrorView(error: posts.error!, onRetry: posts.refresh));
            if (posts.items.isEmpty) {
              return SliverToBoxAdapter(
                child: EmptyView(
                  icon: Sym.photoCamera,
                  title: 'هنوز پستی منتشر نشده',
                  action: _isMe ? PillButton(label: 'ایجاد اولین پست', icon: Sym.addCircle, onTap: () => context.push('/post/new')) : null,
                ),
              );
            }
            return SliverGrid.builder(
              gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: 3, mainAxisSpacing: 2, crossAxisSpacing: 2),
              itemCount: posts.items.length + (posts.done ? 0 : 1),
              itemBuilder: (context, i) {
                if (i >= posts.items.length) {
                  Future.microtask(posts.more);
                  return ColoredBox(color: sk.media);
                }
                final post = posts.items[i];
                return GestureDetector(
                  key: Key('grid-${post.id}'),
                  onTap: () async {
                    await context.push('/post/${post.id}');
                    posts.refresh();
                  },
                  child: Stack(fit: StackFit.expand, children: [
                    ColoredBox(color: sk.panel, child: post.media.isEmpty ? null : NetImage(post.media.first.url, placeholder: sk.panel)),
                    if (post.media.length > 1) const Positioned(top: 6, right: 6, child: Sy(Sym.collections, size: 16, color: Colors.white)),
                  ]),
                );
              },
            );
          },
        ),
      const SliverToBoxAdapter(child: SizedBox(height: 24)),
    ]);
  }

  Widget _actions(BuildContext context, Profile p) {
    final sk = context.skin;
    final r = p.relationship;
    if (_isMe) {
      return Row(children: [
        Expanded(child: _Btn(key: const Key('edit-profile'), icon: Sym.edit, label: 'ویرایش نمایه', onTap: () async {
          await context.push('/profile/edit');
          _load();
        })),
        const SizedBox(width: 8),
        Expanded(child: _Btn(key: const Key('open-settings'), icon: Sym.settings, label: 'تنظیمات', onTap: () => context.push('/settings'))),
        const SizedBox(width: 8),
        _Btn(icon: Sym.group, label: null, semantic: 'درخواست‌های دنبال کردن', onTap: () => context.push('/activity')),
      ]);
    }
    final followLabel = r.following ? 'دنبال می‌کنید' : (r.requested ? 'درخواست ارسال شد' : (r.followsYou ? 'دنبال کردن متقابل' : 'دنبال کردن'));
    return Row(children: [
      Expanded(
        flex: 3,
        child: r.blocking
            ? _Btn(icon: Sym.block, label: 'مسدود شده', onTap: _more)
            : _Btn(
                key: const Key('follow-button'),
                icon: r.following || r.requested ? Sym.check : Sym.personAdd,
                label: followLabel,
                primary: !r.following && !r.requested,
                accent: sk.accent,
                onTap: _busy ? null : _followToggle,
              ),
      ),
      const SizedBox(width: 8),
      Expanded(flex: 2, child: _Btn(icon: Sym.forum, label: 'پیام', onTap: () => soon(context, 'پیام مستقیم', 'فاز ۵'))),
      const SizedBox(width: 8),
      _Btn(key: const Key('profile-more'), icon: Sym.moreHoriz, label: null, semantic: 'گزینه‌های بیشتر', onTap: _more),
    ]);
  }
}

class _Stat extends StatelessWidget {
  const _Stat({super.key, required this.value, required this.label, this.onTap});
  final String value, label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    return GestureDetector(
      onTap: onTap,
      child: SizedBox(
        width: 64,
        child: Column(children: [
          Text(value, style: tw(17, 25.5, 700, sk.text)),
          Text(label, style: tw(11, 16.5, 500, sk.muted), maxLines: 1, overflow: TextOverflow.visible, softWrap: false),
        ]),
      ),
    );
  }
}

/// `h-9 rounded-xl bg-[#F7F8FA] border-[#E6E8ED]` profile action button.
class _Btn extends StatelessWidget {
  const _Btn({super.key, required this.icon, required this.label, required this.onTap, this.semantic, this.primary = false, this.accent});
  final IconData icon;
  final String? label, semantic;
  final VoidCallback? onTap;
  final bool primary;
  final Color? accent;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    final fg = primary ? Colors.white : sk.text;
    return Semantics(
      button: true,
      label: semantic ?? label,
      child: Material(
        color: primary ? (accent ?? sk.accent) : sk.panel,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: primary || sk.cardBorder == null ? BorderSide.none : BorderSide(color: sk.cardBorder!)),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onTap,
          child: SizedBox(
            height: 36,
            width: label == null ? 53 : null,
            child: ExcludeSemantics(
              child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Sy(icon, size: label == null ? 19 : 17, color: fg),
                if (label != null) ...[const SizedBox(width: 4), Flexible(child: Text(label!, style: tw(12, 18, 700, fg), overflow: TextOverflow.ellipsis))],
              ]),
            ),
          ),
        ),
      ),
    );
  }
}

class _Tab extends StatelessWidget {
  const _Tab({required this.icon, required this.active, required this.onTap});
  final IconData icon;
  final bool active;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    return Expanded(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: Stack(alignment: Alignment.center, children: [
          Sy(icon, size: 22, color: active ? sk.accent : sk.muted),
          if (active)
            Positioned(
              bottom: 0,
              child: Container(width: 57, height: 2, decoration: BoxDecoration(color: sk.accent, borderRadius: BorderRadius.circular(2))),
            ),
        ]),
      ),
    );
  }
}
