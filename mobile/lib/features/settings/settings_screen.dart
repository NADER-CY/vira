import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/models.dart';
import '../../core/config/env.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';

/// Settings. Reference: `account_settings_light` (M-232): profile card, appearance (light / dark / system),
/// account & security, purchase & payment, privacy & notifications, support, sign-out and delete links, footer.
/// The dark `_64` draws a different layout (conflict OD-P2-07); this layout is used for both themes.
/// Wallet stays locked behind gate G1; KYC (P4), addresses (P7), notifications (P5) and support (P7) announce
/// their phase. No sample balances, levels or memberships are shown.
class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  Profile? _p;
  Account? _a;
  int? _devices;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final repo = ref.read(repoProvider);
    try {
      final r = await Future.wait([repo.myProfile(), repo.account(), repo.sessions()]);
      if (!mounted) return;
      setState(() {
        _p = r[0] as Profile;
        _a = r[1] as Account;
        _devices = (r[2] as List<DeviceSession>).length;
      });
    } catch (_) {
      // The page stays usable offline; the profile card shows placeholders.
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final mode = ref.watch(themeModeProvider);
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'تنظیمات',
        horizontal: 12,
        backgroundAlpha: 0.8,
        titleStyle: tw(18, 24, 700, cs.onSurface),
        actions: [
          RoundIconButton(icon: Sym.share, label: 'اشتراک‌گذاری نمایه', iconSize: 22, onTap: () => soon(context, 'اشتراک‌گذاری نمایه', 'فاز ۳')),
          HeaderPersonChip(onTap: () => context.go('/profile')),
        ],
      ),
      body: ListView(padding: const EdgeInsets.fromLTRB(16, 8, 16, 32), children: [
        _profileCard(cs),
        const SizedBox(height: 16),
        _appearance(cs, mode),
        const SizedBox(height: 16),
        _group(cs, 'حساب و امنیت', [
          _Row(icon: Sym.badge, title: 'ویرایش مشخصات فردی', subtitle: 'نام، نام کاربری، بیوگرافی و تصویر', onTap: () => context.push('/profile/edit')),
          _Row(
            key: const Key('row-security'),
            icon: Sym.security,
            title: 'ورود دومرحله‌ای و نشست‌های فعال',
            subtitle: _devices == null ? 'مدیریت دستگاه‌های واردشده' : 'فعال روی ${toPersianDigits(_devices!)} دستگاه',
            subtitleColor: cs.tertiary,
            onTap: () => context.push('/settings/security'),
          ),
          _Row(icon: Sym.verifiedUser, title: 'احراز هویت خریدار', subtitle: 'احراز هویت در فاز ۴ فعال می‌شود', subtitleColor: cs.primary, chevronColor: cs.primary, onTap: () => soon(context, 'احراز هویت', 'فاز ۴')),
        ]),
        const SizedBox(height: 16),
        _group(cs, 'خرید و پرداخت', [
          _Row(icon: Sym.locationOn, title: 'آدرس‌های منتخب من', subtitle: 'در فاز ۷ فعال می‌شود', onTap: () => soon(context, 'آدرس‌ها', 'فاز ۷')),
          _Row(icon: Sym.accountBalanceWallet, title: 'کیف پول و کارت‌ها', subtitle: 'قفل تا تأیید دروازهٔ G1', trailingLock: true, onTap: () => toast(context, 'کیف پول تا عبور از دروازهٔ G1 (تأیید حقوقی، امنیتی و تأمین‌کننده) قفل است.')),
        ]),
        const SizedBox(height: 16),
        _group(cs, 'حریم خصوصی و اعلان‌ها', [
          _Row(key: const Key('row-privacy'), icon: Sym.visibility, title: 'حریم خصوصی پیج و استوری', subtitle: 'حساب خصوصی، نظرات، پسندها و مسدودها', onTap: () => context.push('/settings/privacy')),
          _Row(
            icon: Sym.notificationsActive,
            title: 'اعلان‌های زنده و تخفیف‌ها',
            subtitle: 'پیام‌های استریم آنلاین و پیشنهاد ویژه',
            trailing: GestureDetector(
              onTap: () => soon(context, 'اعلان‌ها', 'فاز ۵'),
              child: const AbsorbPointer(child: ArchiveSwitch(value: false, onChanged: null, label: 'اعلان‌های زنده', width: 44, height: 24)),
            ),
          ),
        ]),
        const SizedBox(height: 16),
        _group(cs, 'پشتیبانی و شرایط', [
          _Row(icon: Sym.helpCenter, title: 'مرکز راهنما و قوانین ویرا', subtitle: 'پرسش‌های متداول و قوانین', onTap: () => soon(context, 'مرکز راهنما', 'فاز ۷')),
          _Row(icon: Sym.supportAgent, title: 'گزارش مشکل و ارسال تیکت', subtitle: 'پشتیبانی در فاز ۷ فعال می‌شود', onTap: () => soon(context, 'تیکت پشتیبانی', 'فاز ۷')),
        ]),
        const SizedBox(height: 24),
        Material(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          shadowColor: Colors.black12,
          elevation: 0.5,
          child: InkWell(
            key: const Key('logout-entry'),
            borderRadius: BorderRadius.circular(16),
            onTap: () => context.push('/settings/account'),
            child: SizedBox(
              height: 50,
              child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Sy(Sym.logout, size: 20, color: cs.primary),
                const SizedBox(width: 4),
                Text('خروج از حساب کاربری', style: tw(16, 22, 600, cs.primary)),
              ]),
            ),
          ),
        ),
        const SizedBox(height: 4),
        InkWell(
          key: const Key('delete-entry'),
          borderRadius: BorderRadius.circular(48),
          onTap: () => context.push('/settings/account'),
          child: SizedBox(
            height: 36,
            child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(Sym.deleteForever, size: 16, color: cs.outline),
              const SizedBox(width: 4),
              Text('درخواست حذف حساب کاربری', style: tw(12, 16, 600, cs.outline)),
            ]),
          ),
        ),
        const SizedBox(height: 24),
        Row(mainAxisAlignment: MainAxisAlignment.center, children: [
          Container(width: 8, height: 8, decoration: BoxDecoration(color: cs.primary, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Text('VIRA Social Commerce', style: tw(12, 16, 600, cs.onSurface)),
        ]),
        const SizedBox(height: 4),
        Text('نسخه ${toPersianDigits(Env.appVersion)} ویرا (نسخهٔ آزمایشی)', style: tw(10, 12, 700, cs.outline), textAlign: TextAlign.center),
      ]),
    );
  }

  Widget _profileCard(ColorScheme cs) {
    final p = _p;
    final a = _a;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(16),
        boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
      ),
      clipBehavior: Clip.antiAlias,
      child: Stack(clipBehavior: Clip.none, children: [
        Positioned(left: -48, top: -48, child: Container(width: 128, height: 128, decoration: BoxDecoration(color: cs.primary.withValues(alpha: 0.05), shape: BoxShape.circle))),
        Row(children: [
          SizedBox(
            width: 64,
            height: 68,
            child: Stack(clipBehavior: Clip.none, children: [
              Container(
                width: 64,
                height: 64,
                decoration: BoxDecoration(color: cs.surfaceContainer, shape: BoxShape.circle, boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))]),
                child: Avatar(author: p?.author, size: 64),
              ),
              if (a?.status == 'active')
                Positioned(
                  right: -4,
                  top: 44,
                  child: Container(
                    width: 24,
                    height: 24,
                    decoration: BoxDecoration(color: cs.tertiaryContainer, shape: BoxShape.circle),
                    alignment: Alignment.center,
                    child: Sy(Sym.verified, size: 14, color: cs.onTertiaryContainer, fill: true),
                  ),
                ),
            ]),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Wrap(crossAxisAlignment: WrapCrossAlignment.center, spacing: 4, children: [
                Text(p?.author.name ?? '…', style: tw(18, 24, 700, cs.onSurface)),
                if (p != null)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(color: cs.secondaryFixed.withValues(alpha: 0.5), borderRadius: BorderRadius.circular(999)),
                    child: Text(isolate('@${p.author.handle}'), style: tw(10, 12, 700, cs.secondary)),
                  ),
              ]),
              const SizedBox(height: 2),
              Row(children: [
                Container(width: 6, height: 6, decoration: BoxDecoration(color: a?.status == 'active' ? cs.tertiaryContainer : cs.error, shape: BoxShape.circle)),
                const SizedBox(width: 4),
                Flexible(child: Text(_statusLabel(a?.status), style: tw(12, 16, 500, a?.status == 'active' ? cs.tertiary : cs.error))),
              ]),
              const SizedBox(height: 4),
              Text(a == null ? '' : toPersianDigits(a.phoneMasked), textDirection: TextDirection.ltr, style: tw(10, 12, 700, cs.outline)),
            ]),
          ),
          RoundIconButton(icon: Sym.edit, label: 'ویرایش نمایه', size: 40, iconSize: 20, background: cs.surfaceContainerLow, onTap: () => context.push('/profile/edit')),
        ]),
      ]),
    );
  }

  static String _statusLabel(String? s) => switch (s) {
        'active' => 'حساب فعال و تأییدشده با شماره موبایل',
        'restricted' => 'حساب محدود شده — امکان انتشار محتوا نیست',
        'deletion_pending' => 'در انتظار حذف حساب',
        null => '',
        _ => 'وضعیت حساب: $s',
      };

  Widget _appearance(ColorScheme cs, ThemeMode mode) => Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
        ),
        child: Column(children: [
          Row(children: [
            Container(
              width: 28,
              height: 28,
              decoration: BoxDecoration(color: cs.primary.withValues(alpha: 0.1), shape: BoxShape.circle),
              alignment: Alignment.center,
              child: Sy(Sym.palette, size: 16, color: cs.primary),
            ),
            const SizedBox(width: 4),
            Expanded(child: Text('تنظیمات ظاهر برنامه', style: tw(16, 22, 600, cs.onSurface))),
            Text('حالت تم', style: tw(10, 12, 600, cs.primary)),
          ]),
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(4),
            decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(48)),
            child: Row(children: [
              for (final (m, icon, label) in const [(ThemeMode.light, Sym.lightMode, 'روشن'), (ThemeMode.dark, Sym.darkMode, 'تیره'), (ThemeMode.system, Sym.hdrAuto, 'سیستم')])
                Expanded(
                  child: Semantics(
                    selected: mode == m,
                    button: true,
                    child: GestureDetector(
                      onTap: () => ref.read(themeModeProvider.notifier).set(m),
                      child: Container(
                        height: 34,
                        decoration: BoxDecoration(
                          color: mode == m ? cs.surfaceContainerLowest : null,
                          borderRadius: BorderRadius.circular(32),
                          boxShadow: mode == m ? const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))] : null,
                        ),
                        child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                          Sy(icon, size: 18, color: mode == m ? cs.primary : cs.outline, fill: mode == m),
                          const SizedBox(width: 6),
                          Text(label, style: tw(12, 16, 600, mode == m ? cs.primary : cs.outline)),
                        ]),
                      ),
                    ),
                  ),
                ),
            ]),
          ),
        ]),
      );

  Widget _group(ColorScheme cs, String title, List<Widget> rows) => Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Padding(padding: const EdgeInsetsDirectional.only(start: 4, bottom: 4), child: Text(title, style: tw(14, 18, 600, cs.outline))),
        Container(
          decoration: BoxDecoration(
            color: cs.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(16),
            boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
          ),
          clipBehavior: Clip.antiAlias,
          child: Column(children: [
            for (var i = 0; i < rows.length; i++) ...[
              if (i > 0) Padding(padding: const EdgeInsets.symmetric(horizontal: 16), child: Container(height: 1, color: cs.surfaceContainer)),
              rows[i],
            ],
          ]),
        ),
      ]);
}

class _Row extends StatelessWidget {
  const _Row({super.key, required this.icon, required this.title, required this.subtitle, this.onTap, this.subtitleColor, this.chevronColor, this.trailing, this.trailingLock = false});
  final IconData icon;
  final String title, subtitle;
  final VoidCallback? onTap;
  final Color? subtitleColor, chevronColor;
  final Widget? trailing;
  final bool trailingLock;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return InkWell(
      onTap: onTap,
      child: SizedBox(
        height: 72,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Row(children: [
            Container(width: 40, height: 40, decoration: BoxDecoration(color: cs.surfaceContainer, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: 20, color: cs.onSurface)),
            const SizedBox(width: 16),
            Expanded(
              child: Column(mainAxisAlignment: MainAxisAlignment.center, crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: tw(16, 22, 600, cs.onSurface), maxLines: 1, overflow: TextOverflow.ellipsis),
                Text(subtitle, style: tw(10, 12, 700, subtitleColor ?? cs.outline), maxLines: 1, overflow: TextOverflow.ellipsis),
              ]),
            ),
            const SizedBox(width: 8),
            trailing ?? (trailingLock ? Sy(Sym.lock, size: 20, color: cs.outline) : Sy(Sym.chevronLeft, size: 20, color: chevronColor ?? cs.outline)),
          ]),
        ),
      ),
    );
  }
}
