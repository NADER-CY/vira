import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/ui/paged.dart';
import '../../core/widgets/brand.dart';
import '../../core/widgets/states.dart';
import '../feed/post_card.dart';

/// Home feed. References: `_33` (dark, M-033) and `home_feed_light` (M-250): header (logo, shop, activity,
/// messages, avatar), For you / Following tabs, live-shop and filter chips, stories tray, post cards, end card.
/// Phase 2 activates the two feeds, the activity entry (follow requests) and the avatar; shop (P4), messages (P5),
/// live shop (P8), filters and stories (P3) keep their place and announce their phase.
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({super.key});

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  bool _following = false;
  late final Map<bool, Paged<Post>> _feeds = {
    false: Paged<Post>((c) => ref.read(repoProvider).feed(following: false, cursor: c)),
    true: Paged<Post>((c) => ref.read(repoProvider).feed(following: true, cursor: c)),
  };
  final _scroll = ScrollController();

  Paged<Post> get _feed => _feeds[_following]!;

  @override
  void initState() {
    super.initState();
    _feed.more();
    _scroll.addListener(() {
      if (_scroll.position.extentAfter < 600) Future.microtask(_feed.more);
    });
  }

  @override
  void dispose() {
    for (final f in _feeds.values) {
      f.dispose();
    }
    _scroll.dispose();
    super.dispose();
  }

  void _tab(bool following) {
    if (following == _following) return;
    setState(() => _following = following);
    if (!_feed.loadedOnce) _feed.more();
  }

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    return Scaffold(
      backgroundColor: sk.page,
      appBar: const FeedHeader(title: 'خانه'),
      body: ListenableBuilder(
        listenable: _feed,
        builder: (context, _) {
          final f = _feed;
          return RefreshIndicator(
            color: sk.accent,
            onRefresh: f.refresh,
            child: CustomScrollView(
              controller: _scroll,
              physics: const AlwaysScrollableScrollPhysics(),
              slivers: [
                SliverToBoxAdapter(child: _TabsRow(following: _following, onTab: _tab)),
                const SliverToBoxAdapter(child: _StoriesTray()),
                if (!f.loadedOnce || (f.loading && f.items.isEmpty))
                  const SliverToBoxAdapter(child: _CardSkeleton())
                else if (f.error != null && f.items.isEmpty)
                  SliverToBoxAdapter(child: _EndCard.error(f.error!, onRetry: f.refresh))
                else if (f.items.isEmpty)
                  SliverToBoxAdapter(
                    child: _EndCard(
                      icon: Sym.dynamicFeed,
                      title: _following ? 'هنوز کسی را دنبال نمی‌کنید' : 'فید شما خالی است',
                      message: _following
                          ? 'حساب‌هایی را که دوست دارید دنبال کنید تا پست‌هایشان اینجا بیاید.'
                          : 'حساب‌ها را دنبال کنید یا اولین پست خود را منتشر کنید.',
                      action: 'ایجاد پست',
                      actionIcon: Sym.addCircle,
                      onAction: () => context.push('/post/new'),
                    ),
                  )
                else ...[
                  SliverList.builder(
                    itemCount: f.items.length,
                    itemBuilder: (_, i) {
                      final p = f.items[i];
                      return Padding(
                        padding: EdgeInsets.only(top: i == 0 ? sk.cardGap : 16),
                        child: PostCard(
                          key: ValueKey(p.id),
                          post: p,
                          onChanged: (n) => f.replace((x) => x.id == n.id, (_) => n),
                          onDeleted: () => f.remove((x) => x.id == p.id),
                        ),
                      );
                    },
                  ),
                  if (f.loading) const SliverToBoxAdapter(child: Padding(padding: EdgeInsets.all(24), child: Center(child: CircularProgressIndicator(strokeWidth: 2)))),
                  if (f.error != null) SliverToBoxAdapter(child: _EndCard.error(f.error!, onRetry: f.more)),
                  if (f.done)
                    SliverToBoxAdapter(
                      child: _EndCard(
                        icon: Sym.celebration,
                        title: 'پست‌های جدید را دیدید!',
                        message: 'برای کشف محصولات بیشتر و تخفیف‌های لحظه‌ای به بخش کاوش و ریلز سر بزنید.',
                        action: 'تازه‌سازی فید',
                        actionIcon: Sym.refresh,
                        onAction: f.refresh,
                      ),
                    ),
                ],
                const SliverToBoxAdapter(child: SizedBox(height: 24)),
              ],
            ),
          );
        },
      ),
    );
  }
}

/// Header shared by the tab roots that the archive draws with it (home `_33`, explore `_26`).
class FeedHeader extends ConsumerWidget implements PreferredSizeWidget {
  const FeedHeader({super.key, required this.title});
  final String title;

  @override
  Size get preferredSize => const Size.fromHeight(56);

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final sk = context.skin;
    final dark = context.isDark;
    final handle = ref.watch(sessionProvider).handle;
    return Container(
      decoration: BoxDecoration(
        color: sk.header,
        border: sk.headerBorder == null ? null : Border(bottom: BorderSide(color: sk.headerBorder!)),
        boxShadow: [dark ? const BoxShadow(color: Color(0x59000000), blurRadius: 12, offset: Offset(0, 1)) : const BoxShadow(color: Color(0x0A000000), blurRadius: 6, offset: Offset(0, 1))],
      ),
      child: SafeArea(
        bottom: false,
        child: SizedBox(
          height: 56,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(children: [
              const ViraWordmark(size: 24),
              const SizedBox(width: 8),
              Text(title, style: tw(18, 24, dark ? 600 : 700, sk.text)),
              const Spacer(),
              // shop — Phase 4 (dark: tinted pill with icon; light: gradient pill with label and "هاب")
              Semantics(
                button: true,
                label: 'فروشگاه',
                child: GestureDetector(
                  onTap: () => soon(context, 'فروشگاه', 'فاز ۴'),
                  child: Container(
                    height: dark ? 36 : 32,
                    padding: EdgeInsets.symmetric(horizontal: dark ? 8 : 12),
                    decoration: BoxDecoration(
                      color: sk.shopBg,
                      gradient: sk.shopGradient == null ? null : LinearGradient(colors: sk.shopGradient!, begin: Alignment.topLeft, end: Alignment.bottomRight),
                      border: dark ? null : Border.all(color: Colors.white.withValues(alpha: 0.2)),
                      borderRadius: BorderRadius.circular(999),
                      boxShadow: dark ? null : const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
                    ),
                    child: ExcludeSemantics(
                      child: Row(mainAxisSize: MainAxisSize.min, children: [
                        Sy(Sym.shoppingBag, size: dark ? 18 : 17, color: sk.shopFg),
                        if (!dark) ...[
                          const SizedBox(width: 6),
                          Text('فروشگاه', style: tw(12, 16, 700, sk.shopFg)),
                          const SizedBox(width: 6),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                            decoration: BoxDecoration(color: Colors.white.withValues(alpha: 0.2), borderRadius: BorderRadius.circular(999)),
                            child: Text('هاب', style: tw(9, 9, 700, sk.shopFg)),
                          ),
                        ],
                      ]),
                    ),
                  ),
                ),
              ),
              SizedBox(width: dark ? 4 : 4),
              RoundIconButton(
                icon: Sym.favorite,
                label: 'فعالیت‌ها',
                size: dark ? 44 : 36,
                iconSize: dark ? 24 : 22,
                color: sk.muted,
                onTap: () => context.push('/activity'),
              ),
              RoundIconButton(
                icon: Sym.send,
                label: 'پیام‌ها',
                size: dark ? 44 : 36,
                iconSize: dark ? 24 : 22,
                color: sk.muted,
                onTap: () => soon(context, 'پیام‌ها', 'فاز ۵'),
              ),
              const SizedBox(width: 4),
              GestureDetector(
                onTap: () => context.go('/profile'),
                child: Semantics(
                  button: true,
                  label: 'نمایه من',
                  child: Container(
                    width: 32,
                    height: 32,
                    padding: EdgeInsets.all(dark ? 1 : 1.5),
                    decoration: BoxDecoration(shape: BoxShape.circle, color: dark ? context.cs.primary.withValues(alpha: 0.4) : sk.accent),
                    child: Avatar(author: handle == null ? null : Author(id: '', handle: handle, displayName: handle), size: 30),
                  ),
                ),
              ),
            ]),
          ),
        ),
      ),
    );
  }
}

class _TabsRow extends StatelessWidget {
  const _TabsRow({required this.following, required this.onTab});
  final bool following;
  final ValueChanged<bool> onTab;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    final dark = context.isDark;
    Widget tab(String label, bool selected, bool value) => Semantics(
          selected: selected,
          button: true,
          child: GestureDetector(
            key: Key(value ? 'tab-following' : 'tab-foryou'),
            onTap: () => onTab(value),
            child: Container(
              padding: EdgeInsets.symmetric(horizontal: selected && dark ? 16 : 14, vertical: 6),
              decoration: BoxDecoration(
                color: selected ? sk.badge : null,
                borderRadius: BorderRadius.circular(999),
                boxShadow: selected ? const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))] : null,
              ),
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                if (selected) ...[
                  Container(width: 6, height: 6, decoration: const BoxDecoration(color: Colors.white, shape: BoxShape.circle)),
                  const SizedBox(width: 6),
                ],
                Text(label, style: tw(12, 16, selected ? (dark ? 600 : 700) : (dark ? 600 : 500), selected ? sk.onAccent : sk.muted)),
              ]),
            ),
          ),
        );
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 6),
      child: Row(children: [
        Container(
          padding: const EdgeInsets.all(4),
          decoration: BoxDecoration(color: sk.segment, borderRadius: BorderRadius.circular(999)),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            tab('برای شما', !following, false),
            SizedBox(width: dark ? 8 : 6),
            tab('دنبال‌شده‌ها', following, true),
          ]),
        ),
        const Spacer(),
        GestureDetector(
          onTap: () => soon(context, 'لایو شاپ', 'فاز ۸'),
          child: Container(
            height: 32,
            padding: const EdgeInsets.symmetric(horizontal: 10),
            decoration: BoxDecoration(
              color: sk.liveBg,
              borderRadius: BorderRadius.circular(999),
              border: sk.liveBorder == null ? null : Border.all(color: sk.liveBorder!),
            ),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              Container(width: 8, height: 8, decoration: BoxDecoration(color: sk.badge, shape: BoxShape.circle)),
              const SizedBox(width: 4),
              Text('لایو شاپ', style: tw(11, 14, dark ? 500 : 700, sk.liveFg)),
            ]),
          ),
        ),
        const SizedBox(width: 6),
        RoundIconButton(
          icon: Sym.tune,
          label: 'فیلترها',
          size: 32,
          iconSize: 18,
          color: sk.muted,
          background: sk.chip,
          border: sk.chipBorder == null ? null : Border.all(color: sk.chipBorder!),
          onTap: () => soon(context, 'فیلتر فید', 'فاز ۳'),
        ),
      ]),
    );
  }
}

/// Stories tray: only the viewer's own "add story" bubble — stories are Phase 3, so no sample stories are shown.
class _StoriesTray extends StatelessWidget {
  const _StoriesTray();

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final sk = context.skin;
    return Container(
      decoration: BoxDecoration(
        color: context.isDark ? null : sk.card,
        border: sk.storyDivider == null ? null : Border(bottom: BorderSide(color: sk.storyDivider!)),
      ),
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
      alignment: AlignmentDirectional.centerStart,
      child: GestureDetector(
        onTap: () => soon(context, 'استوری', 'فاز ۳'),
        child: SizedBox(
          width: 64,
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            SizedBox(
              width: 64,
              height: 64,
              child: Stack(children: [
                Container(
                  padding: const EdgeInsets.all(2),
                  decoration: BoxDecoration(color: cs.surfaceContainerHigh, shape: BoxShape.circle),
                  child: Container(decoration: BoxDecoration(color: cs.surfaceContainerHighest, shape: BoxShape.circle), child: Center(child: Sy(Sym.person, size: 28, color: sk.muted))),
                ),
                Positioned(
                  left: 1,
                  bottom: 0,
                  child: Container(
                    width: 19,
                    height: 19,
                    decoration: BoxDecoration(
                      color: cs.primaryContainer,
                      shape: BoxShape.circle,
                      boxShadow: const [BoxShadow(color: Color(0x1A000000), blurRadius: 6, offset: Offset(0, 4), spreadRadius: -1)],
                    ),
                    alignment: Alignment.center,
                    child: Sy(Sym.add, size: 14, color: cs.onPrimaryContainer),
                  ),
                ),
              ]),
            ),
            const SizedBox(height: 6),
            Text('استوری شما', style: tw(11, 14, 500, sk.muted), maxLines: 1, overflow: TextOverflow.ellipsis),
          ]),
        ),
      ),
    );
  }
}

/// The reference's closing card (`پست‌های جدید را دیدید!`): icon disc, title, text, pill button. Reused for the empty
/// and error states so they keep the archive look.
class _EndCard extends StatelessWidget {
  const _EndCard({required this.icon, required this.title, required this.message, required this.action, required this.actionIcon, required this.onAction});

  factory _EndCard.error(Object e, {required Future<void> Function() onRetry}) => _EndCard(
        icon: Sym.cloudOff,
        title: errorTitle(e),
        message: errorDetail(e) ?? 'دوباره تلاش کنید.',
        action: 'تلاش دوباره',
        actionIcon: Sym.refresh,
        onAction: onRetry,
      );

  final IconData icon, actionIcon;
  final String title, message, action;
  final Future<void> Function() onAction;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    return Container(
      margin: const EdgeInsets.only(top: 24),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 24),
      decoration: BoxDecoration(
        color: sk.endCard,
        borderRadius: BorderRadius.circular(16),
        border: sk.endCardBorder == null ? null : Border.all(color: sk.endCardBorder!),
      ),
      child: Column(children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(color: context.cs.primaryContainer.withValues(alpha: 0.2), shape: BoxShape.circle),
          alignment: Alignment.center,
          child: Sy(icon, size: 26, color: sk.accent),
        ),
        const SizedBox(height: 8),
        Text(title, style: tw(18, 24, 700, sk.text), textAlign: TextAlign.center),
        const SizedBox(height: 4),
        ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 260),
          child: Text(message, style: tw(12, 18, 400, sk.muted), textAlign: TextAlign.center),
        ),
        const SizedBox(height: 14),
        Material(
          color: sk.panel == sk.endCard ? context.cs.surfaceContainerHigh : sk.panel,
          borderRadius: BorderRadius.circular(999),
          child: InkWell(
            borderRadius: BorderRadius.circular(999),
            onTap: onAction,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                Sy(actionIcon, size: 18, color: sk.text),
                const SizedBox(width: 4),
                Text(action, style: tw(12, 16, 600, sk.text)),
              ]),
            ),
          ),
        ),
      ]),
    );
  }
}

/// Loading state: the post card's shape with neutral blocks.
class _CardSkeleton extends StatelessWidget {
  const _CardSkeleton();

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    Widget bar(double w, double h) => Container(width: w, height: h, decoration: BoxDecoration(color: sk.media, borderRadius: BorderRadius.circular(6)));
    return Semantics(
      label: 'در حال بارگذاری',
      child: Container(
        margin: EdgeInsets.only(top: sk.cardGap),
        decoration: BoxDecoration(
          color: sk.card,
          borderRadius: BorderRadius.circular(16),
          border: sk.cardBorder == null ? null : Border.all(color: sk.cardBorder!),
        ),
        clipBehavior: Clip.antiAlias,
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Padding(
            padding: const EdgeInsets.all(14),
            child: Row(children: [
              Container(width: 40, height: 40, decoration: BoxDecoration(color: sk.media, shape: BoxShape.circle)),
              const SizedBox(width: 10),
              Column(crossAxisAlignment: CrossAxisAlignment.start, children: [bar(96, 12), const SizedBox(height: 6), bar(64, 10)]),
            ]),
          ),
          AspectRatio(aspectRatio: 4 / 5, child: ColoredBox(color: sk.media)),
          Padding(padding: const EdgeInsets.all(14), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [bar(160, 12), const SizedBox(height: 8), bar(240, 10)])),
        ]),
      ),
    );
  }
}
