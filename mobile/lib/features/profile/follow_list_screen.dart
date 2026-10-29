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

/// Followers / following. Reference: `_109` (M-109, dark; label "Followers Following" localised): account line,
/// two count tabs, search, sort line, person rows with follow / followed / remove buttons. "Online now",
/// verification badges and the suggestions block come with later phases.
class FollowListScreen extends ConsumerStatefulWidget {
  const FollowListScreen({super.key, required this.userId, this.following = false});
  final String userId;
  final bool following;

  @override
  ConsumerState<FollowListScreen> createState() => _FollowListScreenState();
}

class _FollowListScreenState extends ConsumerState<FollowListScreen> {
  late bool _following = widget.following;
  Profile? _p;
  late final Map<bool, Paged<ListEntry>> _lists = {
    false: Paged<ListEntry>((c) => ref.read(repoProvider).followers(widget.userId, cursor: c)),
    true: Paged<ListEntry>((c) => ref.read(repoProvider).following(widget.userId, cursor: c)),
  };
  final _q = TextEditingController();
  final Set<String> _busy = {};

  bool get _mine => widget.userId == ref.read(sessionProvider).userId;
  Paged<ListEntry> get _list => _lists[_following]!;

  @override
  void initState() {
    super.initState();
    _list.more();
    ref.read(repoProvider).user(widget.userId).then((p) {
      if (mounted) setState(() => _p = p);
    }, onError: (Object _) {});
  }

  @override
  void dispose() {
    for (final l in _lists.values) {
      l.dispose();
    }
    _q.dispose();
    super.dispose();
  }

  void _tab(bool following) {
    setState(() => _following = following);
    if (!_list.loadedOnce) _list.more();
  }

  Future<void> _toggle(ListEntry e) async {
    setState(() => _busy.add(e.author.id));
    final repo = ref.read(repoProvider);
    try {
      if (e.following) {
        await repo.unfollow(e.author.id);
      } else {
        final state = await repo.follow(e.author.id);
        if (state == 'requested' && mounted) toast(context, 'درخواست دنبال کردن ارسال شد.');
      }
      final now = !e.following;
      _list.replace((x) => x.author.id == e.author.id, (x) => ListEntry(author: x.author, following: now, since: x.since));
    } on ApiException catch (err) {
      if (mounted) toast(context, err.title);
    } finally {
      if (mounted) setState(() => _busy.remove(e.author.id));
    }
  }

  Future<void> _remove(ListEntry e) async {
    if (!await confirm(context, title: 'حذف دنبال‌کننده', message: '${e.author.name} از دنبال‌کنندگان شما حذف می‌شود و به او اطلاع داده نمی‌شود.', action: 'حذف')) return;
    try {
      await ref.read(repoProvider).removeFollower(e.author.id);
      _list.remove((x) => x.author.id == e.author.id);
    } on ApiException catch (err) {
      if (mounted) toast(context, err.title);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final p = _p;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'دنبال‌کنندگان و دنبال‌شوندگان',
        horizontal: 16,
        shadow: const BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 1)),
        actions: [
          RoundIconButton(icon: Sym.moreVert, label: 'گزینه‌های بیشتر', iconSize: 20, onTap: () => context.push('/settings/privacy')),
          HeaderPersonChip(onTap: () => context.go('/profile')),
        ],
      ),
      body: ListView(padding: const EdgeInsets.fromLTRB(12, 0, 12, 24), children: [
        Row(children: [
          Container(width: 8, height: 8, decoration: BoxDecoration(color: cs.primary, shape: BoxShape.circle)),
          const SizedBox(width: 4),
          Expanded(child: Text('حساب کاربری: ${p == null ? '' : isolate(p.author.handle)}', style: tw(12, 16, 500, cs.onSurfaceVariant))),
        ]),
        const SizedBox(height: 12),
        Container(
          padding: const EdgeInsets.all(4),
          decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(12)),
          child: Row(children: [
            _tabButton(cs, false, p == null ? null : p.stats.followers, 'دنبال‌کننده'),
            _tabButton(cs, true, p == null ? null : p.stats.following, 'دنبال‌شونده'),
          ]),
        ),
        const SizedBox(height: 12),
        Container(
          height: 44,
          decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(12)),
          child: Row(children: [
            const SizedBox(width: 6),
            Sy(Sym.search, size: 20, color: cs.onSurfaceVariant),
            const SizedBox(width: 6),
            Expanded(
              child: TextField(
                controller: _q,
                onChanged: (_) => setState(() {}),
                style: tw(14, 20, 400, cs.onSurface),
                decoration: InputDecoration(
                  isCollapsed: true,
                  border: InputBorder.none,
                  hintText: _following ? 'جستجو در میان دنبال‌شوندگان...' : 'جستجو در میان دنبال‌کنندگان...',
                  hintStyle: tw(14, 20, 400, cs.onSurfaceVariant.withValues(alpha: 0.7)),
                ),
              ),
            ),
          ]),
        ),
        const SizedBox(height: 16),
        Row(children: [
          Expanded(child: Text('مرتب‌سازی بر اساس: جدیدترین‌ها', style: tw(11, 14, 500, cs.onSurfaceVariant))),
          GestureDetector(
            onTap: () => soon(context, 'فیلتر فهرست', 'نسخه‌های بعد'),
            child: Row(children: [Sy(Sym.tune, size: 16, color: cs.onSurfaceVariant), const SizedBox(width: 2), Text('فیلتر', style: tw(11, 14, 500, cs.onSurfaceVariant))]),
          ),
        ]),
        const SizedBox(height: 8),
        ListenableBuilder(
          listenable: _list,
          builder: (context, _) {
            final l = _list;
            if (!l.loadedOnce) return const LoadingView(rows: 4);
            if (l.error != null && l.items.isEmpty) return ErrorView(error: l.error!, onRetry: l.refresh);
            final q = _q.text.trim().toLowerCase();
            final items = q.isEmpty ? l.items : l.items.where((e) => e.author.handle.toLowerCase().contains(q) || e.author.displayName.toLowerCase().contains(q)).toList();
            if (items.isEmpty) {
              return EmptyView(icon: Sym.group, title: _following ? 'هنوز کسی دنبال نشده' : 'هنوز دنبال‌کننده‌ای ندارد');
            }
            return Column(children: [
              for (final e in items) ...[_row(cs, e), const SizedBox(height: 8)],
              if (!l.done)
                TextButton(onPressed: l.loading ? null : l.more, child: Text(l.loading ? 'در حال بارگذاری…' : 'نمایش بیشتر', style: tw(12, 16, 600, cs.primary))),
            ]);
          },
        ),
      ]),
    );
  }

  Widget _tabButton(ColorScheme cs, bool following, int? count, String label) {
    final sel = _following == following;
    return Expanded(
      child: GestureDetector(
        key: Key(following ? 'tab-following-list' : 'tab-followers-list'),
        onTap: () => _tab(following),
        child: SizedBox(
          height: 36,
          child: Stack(alignment: Alignment.center, children: [
            Text.rich(TextSpan(children: [
              if (count != null) TextSpan(text: '${formatNumber(count)} ', style: tw(14, 20, sel ? 700 : 600, sel ? cs.onSurface : cs.onSurfaceVariant)),
              TextSpan(text: label, style: tw(12, 16, 600, cs.onSurfaceVariant)),
            ])),
            if (sel)
              Positioned(
                bottom: 0,
                child: Container(
                  width: 82,
                  height: 3,
                  decoration: BoxDecoration(color: cs.primary, borderRadius: BorderRadius.circular(999), boxShadow: [BoxShadow(color: cs.primary.withValues(alpha: 0.6), blurRadius: 8)]),
                ),
              ),
          ]),
        ),
      ),
    );
  }

  Widget _row(ColorScheme cs, ListEntry e) {
    final me = e.author.id == ref.read(sessionProvider).userId;
    return Container(
      key: Key('person-${e.author.id}'),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(12)),
      child: Row(children: [
        GestureDetector(onTap: () => context.push('/u/${e.author.id}'), child: Avatar(author: e.author, size: 48)),
        const SizedBox(width: 12),
        Expanded(
          child: GestureDetector(
            onTap: () => context.push('/u/${e.author.id}'),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Flexible(child: Text(e.author.name, style: tw(14, 20, 700, cs.onSurface), overflow: TextOverflow.ellipsis)),
                if (e.author.isPrivate) ...[const SizedBox(width: 4), Sy(Sym.lock, size: 14, color: cs.onSurfaceVariant)],
              ]),
              Text(isolate('@${e.author.handle}'), style: tw(12, 18, 400, cs.onSurfaceVariant)),
            ]),
          ),
        ),
        if (!me) ...[
          if (_mine && !_following) ...[
            _btn(cs, 'حذف', cs.surfaceContainerHigh, cs.onSurface, () => _remove(e)),
            const SizedBox(width: 4),
          ],
          _btn(
            cs,
            e.following ? 'دنبال‌شده' : 'دنبال کردن',
            e.following ? cs.surfaceContainerHighest : cs.primaryContainer,
            e.following ? cs.onSurface : cs.onPrimaryFixed,
            _busy.contains(e.author.id) ? null : () => _toggle(e),
            glow: !e.following,
            key: Key('follow-toggle-${e.author.id}'),
          ),
        ],
      ]),
    );
  }

  Widget _btn(ColorScheme cs, String label, Color bg, Color fg, VoidCallback? onTap, {bool glow = false, Key? key}) => GestureDetector(
        key: key,
        onTap: onTap,
        child: Container(
          height: 28,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: bg,
            borderRadius: BorderRadius.circular(8),
            boxShadow: glow ? [BoxShadow(color: cs.primaryContainer.withValues(alpha: 0.3), blurRadius: 10, offset: const Offset(0, 2))] : null,
          ),
          child: Text(label, style: tw(12, 16, glow ? 700 : 600, fg)),
        ),
      );
}
