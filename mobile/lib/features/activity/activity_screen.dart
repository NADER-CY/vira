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

/// Activity centre. Reference: `_12` (M-012, dark): header, title with count, read-all / filter, tabs (all /
/// orders), follow-requests card with approve / delete, then the notification stream. Phase 2 activates the
/// follow requests (private accounts); the notification stream (P5) and orders tab (P7) announce their phase.
class ActivityScreen extends ConsumerStatefulWidget {
  const ActivityScreen({super.key});

  @override
  ConsumerState<ActivityScreen> createState() => _ActivityScreenState();
}

class _ActivityScreenState extends ConsumerState<ActivityScreen> {
  late final Paged<ListEntry> _requests = Paged<ListEntry>((c) => ref.read(repoProvider).followRequests(cursor: c))..more();
  final Set<String> _busy = {};

  @override
  void dispose() {
    _requests.dispose();
    super.dispose();
  }

  Future<void> _decide(ListEntry e, bool approve) async {
    setState(() => _busy.add(e.author.id));
    try {
      final repo = ref.read(repoProvider);
      approve ? await repo.approveRequest(e.author.id) : await repo.declineRequest(e.author.id);
      _requests.remove((x) => x.author.id == e.author.id);
      if (mounted) toast(context, approve ? '${e.author.name} اکنون شما را دنبال می‌کند.' : 'درخواست حذف شد.');
    } on ApiException catch (err) {
      if (mounted) toast(context, err.title);
    } finally {
      if (mounted) setState(() => _busy.remove(e.author.id));
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: _ActivityHeader(cs: cs),
      body: ListenableBuilder(
        listenable: _requests,
        builder: (context, _) {
          final r = _requests;
          return RefreshIndicator(
            onRefresh: r.refresh,
            child: ListView(padding: const EdgeInsets.fromLTRB(12, 16, 12, 24), children: [
              Row(children: [
                Text('مرکز فعالیت‌ها', style: tw(18, 24, 600, cs.onSurface)),
                const SizedBox(width: 4),
                if (r.items.isNotEmpty)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(color: cs.primaryContainer, borderRadius: BorderRadius.circular(999)),
                    child: Text(toPersianDigits(r.items.length), style: tw(11, 14, 700, cs.onPrimary)),
                  ),
                const Spacer(),
                _smallBtn(cs, Sym.doneAll, 'خواندن همه', () => soon(context, 'اعلان‌ها', 'فاز ۵')),
                const SizedBox(width: 4),
                _smallBtn(cs, Sym.tune, null, () => soon(context, 'فیلتر اعلان‌ها', 'فاز ۵')),
              ]),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(4),
                decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(12)),
                child: Row(children: [
                  Expanded(child: _tab(cs, Sym.notificationsActive, 'همه فعالیت‌ها', true, () {})),
                  Expanded(child: _tab(cs, Sym.shoppingBag, 'سفارش‌ها و خرید', false, () => soon(context, 'سفارش‌ها', 'فاز ۷'))),
                ]),
              ),
              const SizedBox(height: 12),
              _requestsCard(cs, r),
              const SizedBox(height: 16),
              const FeatureNotYetView(icon: Sym.notifications, title: 'اعلان‌های پسند، نظر و لایو', phase: 'فاز ۵'),
            ]),
          );
        },
      ),
    );
  }

  Widget _requestsCard(ColorScheme cs, Paged<ListEntry> r) {
    if (!r.loadedOnce) return const LoadingView(rows: 2);
    if (r.error != null && r.items.isEmpty) return ErrorView(error: r.error!, onRetry: r.refresh);
    final shown = r.items.length > 3 ? 3 : r.items.length;
    return Container(
      key: const Key('follow-requests'),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: cs.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(16),
        boxShadow: const [
          BoxShadow(color: Color(0x1A000000), blurRadius: 6, offset: Offset(0, 4), spreadRadius: -1),
          BoxShadow(color: Color(0x1A000000), blurRadius: 4, offset: Offset(0, 2), spreadRadius: -2),
        ],
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Row(children: [
          SizedBox(
            width: shown == 0 ? 32 : 32 + 24.0 * (shown - 1),
            height: 32,
            child: Stack(children: [
              if (r.items.isEmpty)
                Container(width: 32, height: 32, decoration: BoxDecoration(color: cs.surfaceContainerHighest, shape: BoxShape.circle), child: Center(child: Sy(Sym.personAdd, size: 18, color: cs.onSurfaceVariant)))
              else
                for (var i = 0; i < shown; i++)
                  PositionedDirectional(
                    start: 24.0 * i,
                    child: Container(
                      decoration: BoxDecoration(shape: BoxShape.circle, border: Border.all(color: cs.surfaceContainerHigh, width: 2)),
                      child: Avatar(author: r.items[i].author, size: 28),
                    ),
                  ),
            ]),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('درخواست‌های دنبال کردن', style: tw(12, 16, 600, cs.onSurface)),
              Text(r.items.isEmpty ? 'درخواست جدیدی ندارید' : '${toPersianDigits(r.items.length)} نفر منتظر تأیید شما هستند', style: tw(12, 18, 400, cs.onSurfaceVariant)),
            ]),
          ),
          if (r.items.isNotEmpty) Container(width: 10, height: 10, decoration: BoxDecoration(color: cs.primaryContainer, shape: BoxShape.circle)),
        ]),
        for (final e in r.items) ...[
          const SizedBox(height: 8),
          Row(key: Key('request-${e.author.id}'), children: [
            GestureDetector(
              onTap: () => context.push('/u/${e.author.id}'),
              child: SizedBox(
                width: 36,
                height: 36,
                child: Stack(clipBehavior: Clip.none, children: [
                  Avatar(author: e.author, size: 36),
                  Positioned(
                    right: -4,
                    bottom: -4,
                    child: Container(
                      width: 14,
                      height: 14,
                      decoration: BoxDecoration(color: cs.secondaryContainer, shape: BoxShape.circle),
                      alignment: Alignment.center,
                      child: Sy(Sym.personAdd, size: 10, color: cs.onSecondaryContainer),
                    ),
                  ),
                ]),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(e.author.name, style: tw(12, 16, 600, cs.onSurface), overflow: TextOverflow.ellipsis),
                Text(isolate('@${e.author.handle}'), style: tw(11, 14, 500, cs.onSurfaceVariant)),
              ]),
            ),
            _decision(cs, 'تأیید', cs.primaryContainer, cs.onPrimary, _busy.contains(e.author.id) ? null : () => _decide(e, true), Key('approve-${e.author.id}')),
            const SizedBox(width: 6),
            _decision(cs, 'حذف', cs.surfaceContainer, cs.onSurfaceVariant, _busy.contains(e.author.id) ? null : () => _decide(e, false), Key('decline-${e.author.id}')),
          ]),
        ],
        if (!r.done && r.items.isNotEmpty)
          TextButton(onPressed: r.loading ? null : r.more, child: Text('نمایش بیشتر', style: tw(12, 16, 600, cs.primary))),
      ]),
    );
  }

  Widget _decision(ColorScheme cs, String label, Color bg, Color fg, VoidCallback? onTap, Key key) => GestureDetector(
        key: key,
        onTap: onTap,
        child: Container(
          height: 28,
          padding: const EdgeInsets.symmetric(horizontal: 8),
          alignment: Alignment.center,
          decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(8)),
          child: Text(label, style: tw(11, 14, 500, fg)),
        ),
      );

  Widget _smallBtn(ColorScheme cs, IconData icon, String? label, VoidCallback onTap) => Material(
        color: cs.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: onTap,
          child: Container(
            height: 32,
            constraints: const BoxConstraints(minWidth: 32),
            padding: EdgeInsets.symmetric(horizontal: label == null ? 7 : 8),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              Sy(icon, size: 18, color: cs.onSurfaceVariant),
              if (label != null) ...[const SizedBox(width: 4), Text(label, style: tw(11, 14, 500, cs.onSurfaceVariant))],
            ]),
          ),
        ),
      );

  Widget _tab(ColorScheme cs, IconData icon, String label, bool sel, VoidCallback onTap) => GestureDetector(
        onTap: onTap,
        child: Container(
          height: 32,
          decoration: BoxDecoration(color: sel ? cs.primaryContainer : null, borderRadius: BorderRadius.circular(8)),
          child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
            Sy(icon, size: 16, color: sel ? cs.onPrimary : cs.onSurfaceVariant),
            const SizedBox(width: 6),
            Text(label, style: tw(12, 16, 600, sel ? cs.onPrimary : cs.onSurfaceVariant)),
          ]),
        ),
      );
}

class _ActivityHeader extends StatelessWidget implements PreferredSizeWidget {
  const _ActivityHeader({required this.cs});
  final ColorScheme cs;

  @override
  Size get preferredSize => const Size.fromHeight(56);

  @override
  Widget build(BuildContext context) => Container(
        decoration: BoxDecoration(color: cs.surface.withValues(alpha: 0.85), boxShadow: const [BoxShadow(color: Color(0x59000000), blurRadius: 12, offset: Offset(0, 1))]),
        child: SafeArea(
          bottom: false,
          child: SizedBox(
            height: 56,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Row(children: [
                if (Navigator.of(context).canPop()) ...[
                  RoundIconButton(icon: Sym.arrowForward, label: 'بازگشت', size: 36, onTap: () => context.pop()),
                  const SizedBox(width: 4),
                ],
                Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(12),
                    gradient: LinearGradient(begin: Alignment.bottomLeft, end: Alignment.topRight, colors: [cs.primaryContainer, cs.secondaryContainer]),
                  ),
                  alignment: Alignment.center,
                  child: const Sy(Sym.localMall, size: 22, color: Colors.white),
                ),
                const SizedBox(width: 8),
                Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text('ویرا', style: tw(18, 22.5, 600, cs.onSurface)),
                  Text('فعالیت‌ها', style: tw(11, 14, 500, cs.onSurfaceVariant)),
                ]),
                const Spacer(),
                Material(
                  color: cs.surfaceContainerHigh,
                  shape: StadiumBorder(side: BorderSide(color: cs.outlineVariant.withValues(alpha: 0.3))),
                  child: InkWell(
                    customBorder: const StadiumBorder(),
                    onTap: () => soon(context, 'فروشگاه', 'فاز ۴'),
                    child: SizedBox(
                      height: 40,
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 9),
                        child: Row(mainAxisSize: MainAxisSize.min, children: [
                          Sy(Sym.storefront, size: 20, color: cs.primary),
                          const SizedBox(width: 6),
                          Text('فروشگاه', style: tw(11, 14, 700, cs.primary)),
                        ]),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 6),
                RoundIconButton(
                  icon: Sym.shoppingCart,
                  label: 'سبد خرید',
                  size: 40,
                  iconSize: 20,
                  background: cs.surfaceContainerHigh,
                  border: Border.all(color: cs.outlineVariant.withValues(alpha: 0.3)),
                  onTap: () => soon(context, 'سبد خرید', 'فاز ۷'),
                ),
                const SizedBox(width: 6),
                RoundIconButton(icon: Sym.favorite, label: 'فعالیت‌ها', size: 40, iconSize: 22, color: cs.primary, fill: true, onTap: () {}),
              ]),
            ),
          ),
        ),
      );
}
