import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';

/// Sign out and account deletion. Reference: `_206` (M-206, light; dark `_53` — OD-P2-09): account card,
/// sign-out card, temporary-freeze card, danger zone (wallet / open orders / 30-day recovery checks, reason,
/// acknowledgement, delete button). Phase 2: sign-out and the deletion request are live; the deletion takes
/// effect after 30 days and signing in again cancels it. Wallet (G1) and orders (P7) do not exist yet, so their
/// checks show that nothing is pending instead of sample balances. Freeze is not offered by the API yet.
class AccountScreen extends ConsumerStatefulWidget {
  const AccountScreen({super.key});

  @override
  ConsumerState<AccountScreen> createState() => _AccountScreenState();
}

class _AccountScreenState extends ConsumerState<AccountScreen> {
  Profile? _p;
  Account? _a;
  bool _ack = false, _busy = false;
  String? _reason;

  static const _reasons = [
    'تجربه کاربری نامناسب یا کندی سامانه',
    'نیاز به تغییر شماره موبایل ثبت شده',
    'استفاده موقت داشتم و نیازی ندارم',
    'نگرانی‌های مربوط به حریم خصوصی',
    'سایر دلایل شخصی',
  ];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final repo = ref.read(repoProvider);
      final p = await repo.myProfile();
      final a = await repo.account();
      if (mounted) {
        setState(() {
          _p = p;
          _a = a;
        });
      }
    } catch (_) {
      // Sign-out still works without the account card.
    }
  }

  Future<void> _logout() async {
    setState(() => _busy = true);
    await ref.read(sessionProvider).signOut();
  }

  Future<void> _delete() async {
    if (!_ack || _busy) return;
    if (!await confirm(context,
        title: 'حذف دائمی حساب',
        message: 'حساب شما پس از ۳۰ روز حذف می‌شود و از همه دستگاه‌ها خارج می‌شوید. ورود دوباره در این مهلت درخواست را لغو می‌کند.',
        action: 'ارسال درخواست حذف',
        danger: true)) {
      return;
    }
    setState(() => _busy = true);
    try {
      final eff = await ref.read(repoProvider).requestDeletion();
      if (mounted) toast(context, 'درخواست حذف ثبت شد${eff == null ? '' : ' — تاریخ اجرا: ${formatJalaliDate(eff)}'}.');
      await ref.read(sessionProvider).signOut(remote: false);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _busy = false);
        toast(context, e.title);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final p = _p;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'تایید خروج و مدیریت حذف حساب کاربری ویرا',
        height: 64,
        horizontal: 12,
        backIcon: Sym.arrowBack,
        backgroundAlpha: 0.9,
        titleStyle: tw(16, 22, 600, cs.onSurface),
        actions: const [HeaderPersonChip()],
      ),
      body: AbsorbPointer(
        absorbing: _busy,
        child: ListView(padding: const EdgeInsets.fromLTRB(16, 16, 16, 24), children: [
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
            child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
              _disc(cs.errorContainer, Sym.security, cs.onErrorContainer, 40, 24),
              const SizedBox(width: 8),
              Expanded(
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text('مدیریت نشست، خروج یا حذف حساب کاربری', style: tw(18, 24, 700, cs.onSurface)),
                  const SizedBox(height: 4),
                  Text.rich(TextSpan(style: tw(14, 20, 400, cs.onSurfaceVariant), children: [
                    const TextSpan(text: 'کاربر فعال: '),
                    TextSpan(text: p?.author.name ?? '…', style: tw(14, 20, 600, cs.onSurface)),
                    if (p != null) TextSpan(text: ' (${isolate('@${p.author.handle}')})'),
                  ])),
                  const SizedBox(height: 8),
                  Row(children: [
                    Sy(Sym.phonelinkLock, size: 16, color: cs.primary),
                    const SizedBox(width: 4),
                    Expanded(child: Text('نشست فعال روی دستگاه جاری — ${toPersianDigits(_a?.phoneMasked ?? '')}', style: tw(12, 16, 600, cs.onSurfaceVariant))),
                  ]),
                ]),
              ),
            ]),
          ),
          const SizedBox(height: 24),
          _card(cs,
              icon: Sym.logout,
              iconBg: cs.surfaceContainerHigh,
              iconColor: cs.onSurface,
              title: 'خروج موقت از این دستگاه',
              en: 'Log Out Device',
              body: 'با خروج از حساب، اطلاعات حساب شما محفوظ می‌ماند و هر زمان می‌توانید با شماره موبایل خود و کد پیامکی دوباره وارد شوید.',
              button: _pill(cs, key: const Key('logout-button'), bg: cs.surfaceContainerHigh, fg: cs.onSurface, icon: Sym.powerSettingsNew, label: 'خروج از حساب در این دستگاه', onTap: _logout)),
          const SizedBox(height: 24),
          _card(cs,
              icon: Sym.pauseCircle,
              iconBg: cs.secondaryFixed,
              iconColor: cs.onSecondaryFixed,
              title: 'غیرفعال‌سازی موقت (تعلیق)',
              en: 'Temporary Account Freeze',
              body: 'پروفایل شما از دید سایر کاربران پنهان می‌شود؛ با ورود دوباره حساب فعال خواهد شد.',
              button: _pill(cs, bg: cs.secondary, fg: cs.onSecondary, icon: Sym.bedtime, label: 'تعلیق موقت حساب کاربری', onTap: () => soon(context, 'تعلیق موقت حساب', 'نسخه‌های بعد'))),
          const SizedBox(height: 24),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(color: cs.errorContainer.withValues(alpha: 0.2), borderRadius: BorderRadius.circular(16), boxShadow: _sm),
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
              Row(children: [
                _disc(cs.error, Sym.warning, cs.onError, 40, 22, fill: true),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Text('حذف دائمی حساب و کلیه داده‌ها', style: tw(18, 24, 700, cs.onSurface)),
                    const SizedBox(height: 4),
                    Text('منطقه حساس امنیتی (Danger Zone)', style: tw(10, 12, 600, cs.error)),
                  ]),
                ),
              ]),
              const SizedBox(height: 16),
              _check(cs, Sym.accountBalanceWallet, cs.error, 'تسویه موجودی کیف پول', 'کیف پول فعال نیست', cs.surfaceContainerHigh, cs.onSurface,
                  'کیف پول تا تأیید دروازهٔ G1 فعال نمی‌شود؛ موجودی‌ای برای تسویه وجود ندارد.'),
              const SizedBox(height: 8),
              _check(cs, Sym.checkCircle, cs.tertiary, 'سفارش‌های در حال انجام', 'تایید شد', cs.tertiaryFixed, cs.onTertiaryFixed,
                  'هیچ سفارش باز، مرسوله ارسالی فعال یا اختلاف مالی ثبت‌شده‌ای وجود ندارد.'),
              const SizedBox(height: 8),
              _check(cs, Sym.historyToggleOff, cs.onSurfaceVariant, 'مهلت بازیابی ۳۰ روزه', 'پروتکل امنیتی', cs.surfaceContainerHigh, cs.onSurface,
                  'پس از ثبت درخواست، ۳۰ روز فرصت دارید با ورود دوباره، درخواست را لغو کنید. پس از این مهلت حساب و محتوای شما حذف می‌شود.'),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(48)),
                child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                  Text('علت تمایل به حذف حساب کاربری را انتخاب کنید:', style: tw(14, 18, 700, cs.onSurface)),
                  const SizedBox(height: 8),
                  Container(
                    height: 48,
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(999)),
                    child: DropdownButtonHideUnderline(
                      child: DropdownButton<String>(
                        value: _reason,
                        isExpanded: true,
                        icon: Sy(Sym.expandMore, size: 20, color: cs.onSurfaceVariant),
                        hint: Text('لطفاً یک دلیل انتخاب کنید...', style: tw(14, 20, 400, cs.onSurface)),
                        dropdownColor: cs.surfaceContainerLowest,
                        items: [for (final r in _reasons) DropdownMenuItem(value: r, child: Text(r, style: tw(14, 20, 400, cs.onSurface)))],
                        onChanged: (v) => setState(() => _reason = v),
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  ArchiveCheckbox(
                    key: const Key('delete-ack'),
                    value: _ack,
                    onChanged: (v) => setState(() => _ack = v),
                    child: Text('متوجه هستم که با پایان مهلت ۳۰ روزه، نمایه، پست‌ها، نظرها و فهرست دنبال‌کنندگانم به‌طور غیرقابل بازگشت حذف می‌شود.',
                        style: tw(14, 22.75, 400, cs.onSurface)),
                  ),
                ]),
              ),
              const SizedBox(height: 16),
              Opacity(
                opacity: _ack ? 1 : 0.4,
                child: _pill(cs, key: const Key('delete-button'), bg: cs.error, fg: cs.onError, icon: Sym.deleteForever, label: 'تایید و ارسال درخواست حذف دائمی حساب', height: 48, iconSize: 20, onTap: _ack ? _delete : null),
              ),
              const SizedBox(height: 12),
              Text('ورود دوباره با کد پیامکی در مهلت ۳۰ روزه، درخواست حذف را لغو می‌کند.', style: tw(10, 12, 700, cs.onSurfaceVariant), textAlign: TextAlign.center),
            ]),
          ),
        ]),
      ),
    );
  }

  static const _sm = [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))];

  Widget _disc(Color bg, IconData icon, Color fg, double size, double iconSize, {bool fill = false}) =>
      Container(width: size, height: size, decoration: BoxDecoration(color: bg, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: iconSize, color: fg, fill: fill));

  Widget _card(ColorScheme cs, {required IconData icon, required Color iconBg, required Color iconColor, required String title, required String en, required String body, required Widget button}) =>
      Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Row(children: [
            _disc(iconBg, icon, iconColor, 36, 20),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: tw(16, 22, 700, cs.onSurface)),
                Text(en, style: tw(10, 12, 700, cs.onSurfaceVariant)),
              ]),
            ),
          ]),
          const SizedBox(height: 8),
          Text(body, style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
          const SizedBox(height: 16),
          button,
        ]),
      );

  Widget _pill(ColorScheme cs, {Key? key, required Color bg, required Color fg, required IconData icon, required String label, required VoidCallback? onTap, double height = 38, double iconSize = 18}) =>
      Material(
        color: bg,
        borderRadius: BorderRadius.circular(999),
        child: InkWell(
          key: key,
          borderRadius: BorderRadius.circular(999),
          onTap: onTap,
          child: SizedBox(
            height: height,
            child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(icon, size: iconSize, color: fg),
              const SizedBox(width: 4),
              Flexible(child: Text(label, style: tw(14, 18, height > 40 ? 700 : 600, fg), overflow: TextOverflow.ellipsis)),
            ]),
          ),
        ),
      );

  Widget _check(ColorScheme cs, IconData icon, Color iconColor, String title, String badge, Color badgeBg, Color badgeFg, String body) => Container(
        padding: const EdgeInsets.all(8),
        decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(24)),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Padding(padding: const EdgeInsets.only(top: 2, right: 4), child: Sy(icon, size: 22, color: iconColor)),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Expanded(child: Text(title, style: tw(14, 18, 700, cs.onSurface))),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(color: badgeBg, borderRadius: BorderRadius.circular(999)),
                  child: Text(badge, style: tw(10, 12, 700, badgeFg)),
                ),
              ]),
              const SizedBox(height: 4),
              Text(body, style: tw(14, 21, 400, cs.onSurfaceVariant)),
            ]),
          ),
        ]),
      );
}
