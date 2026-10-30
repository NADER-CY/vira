import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/states.dart';

/// Privacy settings. Reference: `_125` (M-125, light) — status banner, private-account card, story/live card,
/// product-tag card, presence card, blocked/restricted card. The dark reference `_46` uses a different layout
/// (conflict OD-P2-06); `_125` is applied to both themes until the owner decides.
/// Phase 2 enforces: private account, comments-from, like-count visibility, blocking. Preferences of later
/// features (story replies P3, product tags P4, presence P5) are stored now and enforced when they ship.
class PrivacyScreen extends ConsumerStatefulWidget {
  const PrivacyScreen({super.key});

  @override
  ConsumerState<PrivacyScreen> createState() => _PrivacyScreenState();
}

class _PrivacyScreenState extends ConsumerState<PrivacyScreen> {
  PrivacySetting? _s;
  List<Author>? _blocked;
  Object? _error;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final repo = ref.read(repoProvider);
      final s = await repo.privacy();
      final b = await repo.blocks();
      if (mounted) {
        setState(() {
          _s = s;
          _blocked = b;
          _error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  Future<void> _save(PrivacySetting next) async {
    final before = _s;
    setState(() {
      _s = next;
      _saving = true;
    });
    try {
      final saved = await ref.read(repoProvider).setPrivacy(next);
      if (mounted) {
        setState(() => _s = saved);
        toast(context, 'تنظیمات با موفقیت ذخیره شد');
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _s = before);
        toast(context, e.title);
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  PrivacySetting _withSetting(String key, Object value) {
    final s = _s!;
    return PrivacySetting(isPrivate: s.isPrivate, commentsFrom: s.commentsFrom, hideLikeCounts: s.hideLikeCounts, settings: {...s.settings, key: value});
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final s = _s;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'تنظیمات حریم خصوصی',
        height: 64,
        horizontal: 16,
        backgroundAlpha: 0.8,
        titleStyle: tw(18, 24, 700, cs.onSurface),
        actions: [HeaderPersonChip(onTap: () => context.go('/profile'))],
      ),
      body: s == null
          ? (_error == null ? const LoadingView(rows: 6) : Center(child: ErrorView(error: _error!, onRetry: _load)))
          : AbsorbPointer(
              absorbing: _saving,
              child: ListView(padding: const EdgeInsets.fromLTRB(16, 0, 16, 24), children: [
                _banner(cs, s),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.lock, iconBg: cs.surfaceContainer, iconColor: cs.onSurface, title: 'حساب کاربری خصوصی', subtitle: 'کنترل نمایش محتوا به عموم',
                    trailing: ArchiveSwitch(key: const Key('private-switch'), value: s.isPrivate, width: 56, height: 32, label: 'حساب خصوصی', onChanged: (v) => _save(s.copyWith(isPrivate: v))),
                    children: [
                      _note(cs, 'هنگامی که حسابتان خصوصی است، فقط افرادی که تایید می‌کنید قادر خواهند بود پست‌ها و دنبال‌کنندگان شما را ببینند. درخواست‌های دنبال کردن در بخش فعالیت‌ها منتظر تأیید شما می‌مانند.'),
                    ]),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.forum, iconBg: cs.secondaryFixed, iconColor: cs.secondary, title: 'نظرات و پسندها', subtitle: 'چه کسانی می‌توانند نظر بدهند', children: [
                  _chips(cs, 'comments', s.commentsFrom, (v) => _save(s.copyWith(commentsFrom: v))),
                  _toggleRow(cs, 'پنهان کردن تعداد پسندها', 'تعداد پسندهای پست‌های شما برای دیگران نمایش داده نمی‌شود',
                      ArchiveSwitch(key: const Key('hide-likes'), value: s.hideLikeCounts, label: 'پنهان کردن تعداد پسندها', width: 48, height: 28, onChanged: (v) => _save(s.copyWith(hideLikeCounts: v)))),
                ]),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.autoStories, iconBg: cs.secondaryFixed, iconColor: cs.secondary, title: 'استوری و لایو استریم', subtitle: 'انتشار تعاملی و مخاطبان اختصاصی', children: [
                  _navRow(cs, Sym.visibilityOff, cs.errorContainer, cs.error, 'مخفی کردن استوری و لایو از', 'افراد انتخاب‌شده استوری را نمی‌بینند', () => soon(context, 'مخفی کردن استوری', 'فاز ۳')),
                  _closeFriends(cs),
                  Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Text('اجازه ارسال پیام و ری‌اکشن', style: tw(16, 22, 600, cs.onSurface)),
                    Text('مشخص کنید چه کسانی می‌توانند روی استوری پاسخ دهند', style: tw(12, 16, 600, cs.onSurfaceVariant)),
                    const SizedBox(height: 8),
                    _chips(cs, 'story', '${s.settings['story_replies_from'] ?? 'everyone'}', (v) => _save(_withSetting('story_replies_from', v))),
                  ]),
                ]),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.shoppingBag, iconBg: cs.primaryFixed, iconColor: cs.primary, title: 'تگ کالا و ویدیوهای خرید', subtitle: 'کنترل پیوندهای تجاری و مشارکتی', children: [
                  _toggleRow(cs, 'تگ شدن در ریلزهای فروشگاهی دیگران', 'به فروشگاه‌ها اجازه دهید به استایل شما ارجاع دهند',
                      ArchiveSwitch(key: const Key('product-tags'), value: '${s.settings['allow_product_tags'] ?? 'true'}' == 'true', width: 48, height: 28, label: 'تگ شدن در ریلزهای فروشگاهی', onChanged: (v) => _save(_withSetting('allow_product_tags', v)))),
                  _toggleRow(cs, 'تایید دستی پیش از نمایش عمومی', 'بررسی تمام تگ‌های محصولات قبل از ثبت در نمایه شما',
                      GestureDetector(onTap: () => soon(context, 'تأیید دستی تگ‌ها', 'فاز ۴'), child: const AbsorbPointer(child: ArchiveSwitch(value: false, width: 48, height: 28, label: 'تایید دستی تگ‌ها', onChanged: null)))),
                  _pendingTags(cs),
                ]),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.sensors, iconBg: cs.surfaceContainerHigh, iconColor: cs.onSurface, title: 'وضعیت حضور و فعالیت', subtitle: 'نمایش آنلاین بودن در تعاملات زنده', children: [
                  _toggleRow(cs, 'نمایش وضعیت آنلاین در دایرکت و اتاق خرید', 'امکان نمایش حضور شما به کاربران فعال در همان لایو شاپینگ',
                      ArchiveSwitch(key: const Key('presence'), value: '${s.settings['show_activity'] ?? 'false'}' == 'true', width: 48, height: 28, label: 'نمایش وضعیت آنلاین', onChanged: (v) => _save(_withSetting('show_activity', v)))),
                ]),
                const SizedBox(height: 24),
                _card(cs, icon: Sym.shieldPerson, iconBg: cs.errorContainer, iconColor: cs.onErrorContainer, title: 'مدیریت مخاطبان مسدود و محدود', subtitle: 'فهرست‌های ایمنی و پیشگیری از مزاحمت', children: [
                  _navRow(cs, Sym.block, null, cs.outline, 'حساب‌های مسدودشده (Blocked)', 'قطع کامل دسترسی به پروفایل و پیام‌ها', _showBlocked,
                      count: _blocked?.length, key: const Key('blocked-row')),
                  _navRow(cs, Sym.doNotDisturbOn, null, cs.outline, 'حساب‌های محدودشده (Restricted)', 'کنترل نظرات بدون نیاز به آنفالو کردن', () => soon(context, 'حساب‌های محدودشده', 'فاز ۵')),
                ]),
                const SizedBox(height: 24),
                Opacity(
                  opacity: 0.7,
                  child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                    Sy(Sym.lockReset, size: 16, color: cs.onSurfaceVariant),
                    const SizedBox(width: 4),
                    Text('رمزنگاری سرتاسری تعاملات فعال است', style: tw(12, 16, 600, cs.onSurfaceVariant)),
                  ]),
                ),
              ]),
            ),
    );
  }

  Widget _banner(ColorScheme cs, PrivacySetting s) => Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(32),
          gradient: LinearGradient(
            begin: Alignment.centerLeft,
            end: Alignment.centerRight,
            colors: [cs.primary.withValues(alpha: 0.1), cs.secondary.withValues(alpha: 0.05), cs.primary.withValues(alpha: 0)],
          ),
        ),
        child: Row(children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: cs.primary,
              shape: BoxShape.circle,
              boxShadow: const [BoxShadow(color: Color(0x1A000000), blurRadius: 6, offset: Offset(0, 4), spreadRadius: -1)],
            ),
            alignment: Alignment.center,
            child: Sy(Sym.verifiedUser, size: 22, color: cs.onPrimary),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('سطح امنیت و حفظ خلوت', style: tw(16, 22, 600, cs.onSurface)),
              Text(s.isPrivate ? 'حساب خصوصی؛ فقط دنبال‌کنندگان تأییدشده' : 'حساب عمومی؛ پست‌ها برای همه قابل مشاهده است', style: tw(14, 20, 400, cs.onSurfaceVariant)),
            ]),
          ),
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: cs.surfaceContainerLowest,
              borderRadius: BorderRadius.circular(999),
              boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
            ),
            child: Text(s.isPrivate ? 'خصوصی' : 'عمومی', style: tw(12, 16, 600, cs.primary)),
          ),
        ]),
      );

  Widget _card(ColorScheme cs, {required IconData icon, required Color iconBg, required Color iconColor, required String title, required String subtitle, Widget? trailing, required List<Widget> children}) =>
      Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(32),
          boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Row(children: [
            Container(width: 36, height: 36, decoration: BoxDecoration(color: iconBg, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: 20, color: iconColor)),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: tw(18, 24, 700, cs.onSurface)),
                const SizedBox(height: 6),
                Text(subtitle, style: tw(12, 16, 600, cs.onSurfaceVariant)),
              ]),
            ),
            ?trailing,
          ]),
          for (final c in children) ...[const SizedBox(height: 16), c],
        ]),
      );

  Widget _note(ColorScheme cs, String text) => Container(
        padding: const EdgeInsets.all(16),
        color: cs.surfaceContainerLow,
        child: Text(text, style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
      );

  Widget _toggleRow(ColorScheme cs, String title, String sub, Widget toggle) => Row(children: [
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(title, style: tw(16, 22, 600, cs.onSurface)),
            Text(sub, style: tw(14, 20, 400, cs.onSurfaceVariant)),
          ]),
        ),
        const SizedBox(width: 20),
        toggle,
      ]);

  Widget _navRow(ColorScheme cs, IconData icon, Color? iconBg, Color iconColor, String title, String sub, VoidCallback onTap, {int? count, Key? key}) => InkWell(
        key: key,
        onTap: onTap,
        borderRadius: BorderRadius.circular(16),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Row(children: [
            if (iconBg != null)
              Container(width: 30, height: 32, decoration: BoxDecoration(color: iconBg, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: 18, color: iconColor))
            else
              Sy(icon, size: 20, color: iconColor),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: tw(16, 22, 600, cs.onSurface)),
                Text(sub, style: tw(iconBg != null ? 14 : 12, iconBg != null ? 20 : 16, iconBg != null ? 400 : 600, cs.onSurfaceVariant)),
              ]),
            ),
            if (count != null)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
                decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(999)),
                child: Text('${_fa(count)} نفر', style: tw(12, 16, 600, cs.onSurface)),
              ),
            const SizedBox(width: 4),
            Sy(Sym.chevronLeft, size: 20, color: cs.outline),
          ]),
        ),
      );

  Widget _closeFriends(ColorScheme cs) => Container(
        padding: const EdgeInsets.all(16),
        color: cs.surfaceContainerLow,
        child: Row(children: [
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(color: cs.tertiaryFixed, shape: BoxShape.circle),
            alignment: Alignment.center,
            child: Sy(Sym.stars, size: 18, color: cs.tertiary, fill: true),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Wrap(crossAxisAlignment: WrapCrossAlignment.center, spacing: 6, children: [
                Text('دوستان نزدیک', style: tw(16, 22, 600, cs.onSurface)),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(color: cs.tertiaryContainer, borderRadius: BorderRadius.circular(999)),
                  child: Text('Close Friends', style: tw(10, 12, 700, cs.onTertiaryContainer)),
                ),
              ]),
              Text('اشتراک محتوای ویژه و اختصاصی', style: tw(12, 16, 600, cs.onSurfaceVariant)),
            ]),
          ),
          GestureDetector(
            onTap: () => soon(context, 'دوستان نزدیک', 'فاز ۳'),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 4),
              decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(999)),
              child: Text('ویرایش', style: tw(12, 16, 600, cs.tertiary)),
            ),
          ),
        ]),
      );

  Widget _pendingTags(ColorScheme cs) => GestureDetector(
        onTap: () => soon(context, 'تگ‌های در انتظار', 'فاز ۴'),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 9),
          color: cs.surfaceContainerLow,
          child: Row(children: [
            Sy(Sym.pendingActions, size: 18, color: cs.primary),
            const SizedBox(width: 4),
            Expanded(child: Text('تگ‌های در انتظار بازبینی و تایید', style: tw(12, 16, 600, cs.onSurface))),
            Sy(Sym.chevronLeft, size: 18, color: cs.outline),
          ]),
        ),
      );

  /// `همه کاربران · دنبال‌کنندگان · هیچ‌کس` — selected chip is inverse-surface.
  Widget _chips(ColorScheme cs, String group, String value, ValueChanged<String> onPick) => Row(children: [
        for (final (v, label) in const [('everyone', 'همه کاربران'), ('followers', 'دنبال‌کنندگان'), ('nobody', 'هیچ‌کس')]) ...[
          if (v != 'everyone') const SizedBox(width: 4),
          Expanded(
            child: Semantics(
              selected: v == value,
              button: true,
              child: GestureDetector(
                key: Key('$group-$v'),
                onTap: () => onPick(v),
                child: Container(
                  height: 32,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: v == value ? cs.inverseSurface : cs.surfaceContainerHigh,
                    borderRadius: BorderRadius.circular(999),
                    boxShadow: v == value ? const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))] : null,
                  ),
                  child: Text(label, style: tw(12, 16, 600, v == value ? cs.onInverseSurface : cs.onSurface)),
                ),
              ),
            ),
          ),
        ],
      ]);

  Future<void> _showBlocked() async {
    final list = _blocked ?? const <Author>[];
    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (ctx) {
        final cs = ctx.cs;
        return SafeArea(
          child: ConstrainedBox(
            constraints: BoxConstraints(maxHeight: MediaQuery.of(ctx).size.height * 0.7),
            child: list.isEmpty
                ? const EmptyView(icon: Sym.block, title: 'حسابی را مسدود نکرده‌اید')
                : ListView(shrinkWrap: true, children: [
                    Padding(padding: const EdgeInsets.only(bottom: 8), child: Text('حساب‌های مسدودشده', style: tw(16, 22, 700, cs.onSurface), textAlign: TextAlign.center)),
                    for (final a in list)
                      ListTile(
                        leading: Avatar(author: a, size: 40),
                        title: Text(a.name, style: tw(14, 20, 600, cs.onSurface)),
                        subtitle: Text('@${a.handle}', style: tw(12, 16, 400, cs.onSurfaceVariant)),
                        trailing: TextButton(
                          onPressed: () async {
                            Navigator.of(ctx).pop();
                            try {
                              await ref.read(repoProvider).unblock(a.id);
                              await _load();
                              if (mounted) toast(context, 'مسدودی برداشته شد.');
                            } on ApiException catch (e) {
                              if (mounted) toast(context, e.title);
                            }
                          },
                          child: Text('رفع مسدودی', style: tw(12, 16, 700, cs.primary)),
                        ),
                      ),
                  ]),
          ),
        );
      },
    );
  }
}

String _fa(int n) => n.toString().split('').map((d) => '۰۱۲۳۴۵۶۷۸۹'[int.parse(d)]).join();
