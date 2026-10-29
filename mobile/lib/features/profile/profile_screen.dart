import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/config/env.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/tokens.g.dart';
import '../../core/widgets/states.dart';

/// Profile and settings. The theme choice works now; other entries show the phase or gate that delivers them.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mode = ref.watch(themeModeProvider);
    final t = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('پروفایل')),
      body: ListView(children: [
        Padding(
          padding: const EdgeInsets.all(ViraTokens.space4),
          child: Row(children: [
            CircleAvatar(radius: 32, backgroundColor: t.colorScheme.primaryContainer,
                child: Icon(Icons.person, size: 32, color: t.colorScheme.onPrimaryContainer)),
            const SizedBox(width: ViraTokens.space4),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('مهمان', style: t.textTheme.titleMedium),
                Text('ورود با شماره موبایل در فاز ۲', style: t.textTheme.bodySmall?.copyWith(color: t.colorScheme.onSurfaceVariant)),
              ]),
            ),
          ]),
        ),
        const Divider(),
        const _Section('ظاهر'),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: ViraTokens.space4),
          child: SegmentedButton<ThemeMode>(
            segments: const [
              ButtonSegment(value: ThemeMode.light, icon: Icon(Icons.light_mode_outlined), label: Text('روشن')),
              ButtonSegment(value: ThemeMode.dark, icon: Icon(Icons.dark_mode_outlined), label: Text('تیره')),
              ButtonSegment(value: ThemeMode.system, icon: Icon(Icons.brightness_auto_outlined), label: Text('سیستم')),
            ],
            selected: {mode},
            onSelectionChanged: (s) => ref.read(themeModeProvider.notifier).set(s.first),
          ),
        ),
        const SizedBox(height: ViraTokens.space2),
        const _Section('حساب و خرید'),
        const _Entry(Icons.receipt_long_outlined, 'سفارش‌های من', 'فاز ۷'),
        const _Entry(Icons.bookmark_border, 'ذخیره‌شده‌ها', 'فاز ۲'),
        const _Entry(Icons.verified_user_outlined, 'احراز هویت', 'فاز ۴'),
        const _Entry(Icons.account_balance_wallet_outlined, 'کیف پول', 'G1', locked: true),
        const _Section('فروشندگان'),
        const _Entry(Icons.storefront_outlined, 'مدیریت غرفهٔ من', 'فاز ۴'),
        const _Section('پشتیبانی'),
        const _Entry(Icons.support_agent_outlined, 'راهنما و تیکت‌ها', 'فاز ۷'),
        Padding(
          padding: const EdgeInsets.all(ViraTokens.space4),
          child: Text('نسخهٔ ${toPersianDigits(Env.appVersion)}', style: t.textTheme.bodySmall, textAlign: TextAlign.center),
        ),
      ]),
    );
  }
}

class _Section extends StatelessWidget {
  const _Section(this.title);
  final String title;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsetsDirectional.fromSTEB(ViraTokens.space4, ViraTokens.space4, ViraTokens.space4, ViraTokens.space2),
        child: Text(title, style: Theme.of(context).textTheme.labelLarge?.copyWith(color: Theme.of(context).colorScheme.primary)),
      );
}

class _Entry extends StatelessWidget {
  const _Entry(this.icon, this.label, this.phase, {this.locked = false});
  final IconData icon;
  final String label;
  final String phase;
  final bool locked;

  @override
  Widget build(BuildContext context) => ListTile(
        leading: Icon(icon),
        title: Text(label),
        trailing: PhaseBadge(phase, locked: locked),
        onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(
          builder: (_) => Scaffold(appBar: AppBar(title: Text(label)), body: Center(child: FeatureNotYetView(title: label, phase: phase, locked: locked))),
        )),
      );
}
