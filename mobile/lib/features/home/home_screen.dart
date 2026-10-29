import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/tokens.g.dart';
import '../../core/widgets/brand.dart';
import '../../core/widgets/states.dart';

/// Home: top bar (wordmark, shop chip, notifications, messages — as drawn in the archive), feed tabs, stories
/// tray and the feed. Feed content arrives in Phase 2; Phase 1 shows the real server connection state.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final t = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        titleSpacing: ViraTokens.space4,
        title: const ViraWordmark(),
        actions: [
          Padding(
            padding: const EdgeInsetsDirectional.only(end: ViraTokens.space1),
            child: ActionChip(
              avatar: Icon(Icons.storefront_outlined, size: 18, color: t.colorScheme.onPrimary),
              label: Text('فروشگاه', style: TextStyle(color: t.colorScheme.onPrimary)),
              backgroundColor: t.colorScheme.primary,
              side: BorderSide.none,
              onPressed: () => _soon(context, 'فروشگاه', 'فاز ۴'),
            ),
          ),
          IconButton(tooltip: 'اعلان‌ها', icon: const Icon(Icons.favorite_border), onPressed: () => _soon(context, 'اعلان‌ها', 'فاز ۵')),
          IconButton(tooltip: 'پیام‌ها', icon: const Icon(Icons.chat_bubble_outline), onPressed: () => _soon(context, 'پیام‌ها', 'فاز ۵')),
          const SizedBox(width: ViraTokens.space2),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () => ref.refresh(clientConfigProvider.future),
        child: ListView(
          padding: const EdgeInsets.all(ViraTokens.space4),
          children: [
            Row(children: [
              ChoiceChip(label: const Text('برای شما'), selected: true, onSelected: (_) {}),
              const SizedBox(width: ViraTokens.space2),
              ChoiceChip(label: const Text('دنبال‌شده‌ها'), selected: false, onSelected: (_) {}),
            ]),
            const SizedBox(height: ViraTokens.space4),
            SizedBox(
              height: 92,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: 6,
                separatorBuilder: (_, _) => const SizedBox(width: ViraTokens.space3),
                itemBuilder: (_, i) => Column(children: [
                  Container(
                    width: 64,
                    height: 64,
                    padding: const EdgeInsets.all(2.5),
                    decoration: const BoxDecoration(shape: BoxShape.circle, gradient: LinearGradient(colors: ViraTokens.gradientBrand)),
                    child: CircleAvatar(
                      backgroundColor: t.colorScheme.surfaceContainerHighest,
                      child: i == 0 ? Icon(Icons.add, color: t.colorScheme.primary) : null,
                    ),
                  ),
                  const SizedBox(height: ViraTokens.space1),
                  Text(i == 0 ? 'استوری شما' : '', style: t.textTheme.labelSmall),
                ]),
              ),
            ),
            const SizedBox(height: ViraTokens.space4),
            const _ServerStatusCard(),
            const Card(
              child: FeatureNotYetView(icon: Icons.dynamic_feed_outlined, title: 'فید پست‌ها و محصولات', phase: 'فاز ۲'),
            ),
          ],
        ),
      ),
    );
  }

  static void _soon(BuildContext context, String what, String phase) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text('$what در $phase فعال می‌شود.')));
  }
}

class _ServerStatusCard extends ConsumerWidget {
  const _ServerStatusCard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final cfg = ref.watch(clientConfigProvider);
    final t = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(ViraTokens.space4),
        child: cfg.when(
          loading: () => const LoadingView(rows: 2),
          error: (e, _) => ErrorView(error: e, onRetry: () => ref.invalidate(clientConfigProvider)),
          data: (c) => Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Icon(Icons.check_circle, color: t.colorScheme.primary),
              const SizedBox(width: ViraTokens.space2),
              Text('اتصال به سرور ویرا برقرار است', style: t.textTheme.titleMedium),
            ]),
            const SizedBox(height: ViraTokens.space2),
            Text(
              '${toPersianDigits(c.regulatedLocked.length)} قابلیت مالی قانون‌مند تا تأیید نهایی قفل است.',
              style: t.textTheme.bodyMedium?.copyWith(color: t.colorScheme.onSurfaceVariant),
            ),
          ]),
        ),
      ),
    );
  }
}
