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
import '../../core/widgets/states.dart';

/// Sign-in security and active sessions. Reference: `_205` (M-205, light; dark `_77` differs — OD-P2-08).
/// Real data: masked phone, this device, other sessions (platform, model, app version, last use), end one /
/// end all others. Security-UI corrections (docs/progress/P2.md §Security UI): the reference claims "SSL v3"
/// and an active 2FA with authenticator and backup codes, none of which is true in Phase 2 — the texts state
/// what is actually in place (TLS transport, SMS one-time code) and the authenticator / backup-code rows are
/// kept but inactive.
class SecurityScreen extends ConsumerStatefulWidget {
  const SecurityScreen({super.key});

  @override
  ConsumerState<SecurityScreen> createState() => _SecurityScreenState();
}

class _SecurityScreenState extends ConsumerState<SecurityScreen> {
  List<DeviceSession>? _sessions;
  Account? _account;
  Object? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final repo = ref.read(repoProvider);
      final s = await repo.sessions();
      final a = await repo.account();
      if (mounted) {
        setState(() {
          _sessions = s;
          _account = a;
          _error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  Future<void> _end(DeviceSession s) async {
    if (!await confirm(context, title: 'خاتمه نشست', message: 'این دستگاه از حساب شما خارج می‌شود.', action: 'خاتمه نشست', danger: true)) return;
    await _run(() => ref.read(repoProvider).revokeSession(s.id), 'نشست خاتمه یافت.');
  }

  Future<void> _endOthers() async {
    if (!await confirm(context, title: 'خاتمه همه نشست‌های دیگر', message: 'همه دستگاه‌ها به جز همین گوشی از حساب شما خارج می‌شوند.', action: 'خاتمه همه', danger: true)) {
      return;
    }
    await _run(() => ref.read(repoProvider).revokeOtherSessions(), 'همه نشست‌های دیگر خاتمه یافت.');
  }

  Future<void> _run(Future<void> Function() op, String done) async {
    setState(() => _busy = true);
    try {
      await op();
      await _load();
      if (mounted) toast(context, done);
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final list = _sessions;
    final current = list?.where((s) => s.current).firstOrNull;
    final others = list?.where((s) => !s.current).toList() ?? const <DeviceSession>[];
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'امنیت ورود و نشست‌های فعال ویرا',
        height: 64,
        horizontal: 16,
        backIcon: Sym.arrowBack,
        titleStyle: tw(16, 20, 600, cs.onSurface),
        actions: [
          RoundIconButton(icon: Sym.shield, label: 'امنیت', iconSize: 22, color: cs.onSurfaceVariant, onTap: () => context.push('/settings/privacy')),
          HeaderPersonChip(onTap: () => context.go('/profile')),
        ],
      ),
      body: list == null
          ? (_error == null ? const LoadingView(rows: 6) : Center(child: ErrorView(error: _error!, onRetry: _load)))
          : AbsorbPointer(
              absorbing: _busy,
              child: ListView(padding: const EdgeInsets.fromLTRB(16, 8, 16, 24), children: [
                Container(
                  height: 34,
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(999)),
                  child: Row(children: [
                    Sy(Sym.verifiedUser, size: 18, color: cs.tertiary, fill: true),
                    const SizedBox(width: 4),
                    Expanded(child: Text('اتصال رمزنگاری‌شده (TLS) فعال', style: tw(12, 16, 600, cs.onSurface))),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(color: cs.surfaceContainerHighest, borderRadius: BorderRadius.circular(999)),
                      child: Text('HTTPS', style: tw(10, 12, 700, cs.onSurfaceVariant)),
                    ),
                  ]),
                ),
                const SizedBox(height: 16),
                _card(cs, padding: 24, child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                  Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Container(width: 48, height: 48, decoration: BoxDecoration(color: cs.tertiaryContainer, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.gppGood, size: 26, color: Colors.white, fill: true)),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text('ورود با کد یک‌بار مصرف فعال است', style: tw(18, 24.75, 700, cs.onSurface)),
                        const SizedBox(height: 4),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                          decoration: BoxDecoration(color: cs.tertiary.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(999)),
                          child: Text('• تأیید با شماره موبایل', style: tw(10, 12, 700, cs.tertiary)),
                        ),
                        const SizedBox(height: 4),
                        Text('هر ورود با کد پیامکی تأیید می‌شود و نشست‌های هر دستگاه را می‌توانید از همین صفحه خاتمه دهید.', style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
                      ]),
                    ),
                  ]),
                  const SizedBox(height: 16),
                  _factor(cs, Sym.sms, 'پیامک اعتبارسنجی (SMS)', Text(toPersianDigits(_account?.phoneMasked ?? ''), textDirection: TextDirection.ltr, textAlign: TextAlign.right, style: tw(14, 20, 400, cs.onSurfaceVariant)),
                      const ArchiveSwitch(value: true, onChanged: null, label: 'پیامک اعتبارسنجی', height: 28)),
                  const SizedBox(height: 8),
                  _factor(cs, Sym.password, 'برنامه احراز هویت', Text('Google Authenticator یا Microsoft', style: tw(14, 20, 400, cs.onSurfaceVariant), maxLines: 1, overflow: TextOverflow.ellipsis),
                      GestureDetector(onTap: () => soon(context, 'برنامه احراز هویت', 'فازهای بعد'), child: const AbsorbPointer(child: ArchiveSwitch(value: false, onChanged: null, label: 'برنامه احراز هویت', height: 28))),
                      badge: 'پیشنهادی امن'),
                  const SizedBox(height: 8),
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
                    child: Column(children: [
                      Row(children: [
                        _iconDisc(cs, Sym.pin),
                        const SizedBox(width: 16),
                        Expanded(
                          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                            Text('کدهای پشتیبان اضطراری', style: tw(16, 22, 600, cs.onSurface)),
                            Text('برای زمان عدم دسترسی به سیم‌کارت', style: tw(14, 20, 400, cs.onSurfaceVariant)),
                          ]),
                        ),
                      ]),
                      const SizedBox(height: 12),
                      Material(
                        color: cs.surfaceContainerLowest,
                        borderRadius: BorderRadius.circular(999),
                        child: InkWell(
                          borderRadius: BorderRadius.circular(999),
                          onTap: () => soon(context, 'کدهای پشتیبان', 'فازهای بعد'),
                          child: SizedBox(
                            height: 34,
                            child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                              Sy(Sym.key, size: 18, color: cs.onSurface),
                              const SizedBox(width: 4),
                              Text('مشاهده یا تولید مجدد کدها', style: tw(14, 18, 600, cs.onSurface)),
                            ]),
                          ),
                        ),
                      ),
                    ]),
                  ),
                ])),
                const SizedBox(height: 24),
                _sectionTitle(cs, 'دستگاه فعلی (نشست جاری)', 'همگام‌سازی لحظه‌ای', cs.tertiary),
                const SizedBox(height: 8),
                if (current != null)
                  _card(cs, padding: 24, child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                    Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Container(width: 48, height: 48, decoration: BoxDecoration(color: cs.primary.withValues(alpha: 0.1), shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.phoneIphone, size: 24, color: cs.primary)),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                          Text(_deviceName(current), style: tw(16, 22, 600, cs.onSurface)),
                          const SizedBox(height: 4),
                          Text('سیستم‌عامل ${_platform(current.platform)}', style: tw(14, 20, 400, cs.onSurfaceVariant)),
                        ]),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(color: cs.tertiary.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(999)),
                        child: Text('این دستگاه (آنلاین)', style: tw(10, 12, 700, cs.tertiary)),
                      ),
                    ]),
                    const SizedBox(height: 16),
                    Container(
                      padding: const EdgeInsets.all(16),
                      decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
                      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        _kv(cs, Sym.schedule, 'ورود:', current.createdAt == null ? '—' : formatJalaliDate(current.createdAt!)),
                        const SizedBox(height: 4),
                        _kv(cs, Sym.apps, 'اپلیکیشن ویرا نسخه', isolate(current.appVersion.isEmpty ? '—' : current.appVersion)),
                      ]),
                    ),
                  ])),
                const SizedBox(height: 24),
                _sectionTitle(cs, 'نشست‌های فعال دیگر', '${toPersianDigits(others.length)} نشست فعال', cs.onSurfaceVariant),
                const SizedBox(height: 8),
                if (others.isEmpty)
                  _card(cs, padding: 16, child: Text('دستگاه دیگری وارد حساب شما نیست.', style: tw(14, 22, 400, cs.onSurfaceVariant), textAlign: TextAlign.center))
                else
                  for (final s in others) ...[
                    _card(cs, padding: 16, child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                      Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        _iconDisc(cs, s.platform == 'web' ? Sym.laptopMac : Sym.smartphone),
                        const SizedBox(width: 16),
                        Expanded(
                          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                            Text(_deviceName(s), style: tw(16, 22, 600, cs.onSurface)),
                            Text('آخرین فعالیت: ${s.lastUsedAt == null ? '—' : formatRelative(s.lastUsedAt!)}', style: tw(12, 16, 600, cs.onSurfaceVariant)),
                          ]),
                        ),
                        const SizedBox(width: 8),
                        GestureDetector(
                          key: Key('end-${s.id}'),
                          onTap: () => _end(s),
                          child: Container(
                            height: 24,
                            padding: const EdgeInsets.symmetric(horizontal: 12),
                            decoration: BoxDecoration(color: cs.errorContainer, borderRadius: BorderRadius.circular(999)),
                            child: Row(mainAxisSize: MainAxisSize.min, children: [
                              Sy(Sym.logout, size: 16, color: cs.onErrorContainer),
                              const SizedBox(width: 2),
                              Text('خاتمه نشست', style: tw(12, 16, 600, cs.onErrorContainer)),
                            ]),
                          ),
                        ),
                      ]),
                      const SizedBox(height: 8),
                      Row(children: [
                        Text('${_platform(s.platform)} ', style: tw(10, 12, 700, cs.onSurfaceVariant)),
                        Text('•', style: tw(16, 24, 400, cs.outlineVariant)),
                        Text(' نسخه ${isolate(s.appVersion.isEmpty ? '—' : s.appVersion)}', style: tw(10, 12, 700, cs.onSurfaceVariant)),
                      ]),
                    ])),
                    const SizedBox(height: 8),
                  ],
                const SizedBox(height: 16),
                Material(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(999),
                  elevation: 0.5,
                  shadowColor: Colors.black12,
                  child: InkWell(
                    key: const Key('end-others'),
                    borderRadius: BorderRadius.circular(999),
                    onTap: others.isEmpty ? null : _endOthers,
                    child: SizedBox(
                      height: 52,
                      child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                        Sy(Sym.deleteSweep, size: 20, color: cs.error.withValues(alpha: others.isEmpty ? 0.4 : 1)),
                        const SizedBox(width: 8),
                        Text('خاتمه دادن به تمام نشست‌های دیگر', style: tw(14, 18, 600, cs.error.withValues(alpha: others.isEmpty ? 0.4 : 1))),
                      ]),
                    ),
                  ),
                ),
                const SizedBox(height: 4),
                Text('با زدن این دکمه، اتصال شما از تمام دستگاه‌ها به جز گوشی فعلی قطع می‌شود و برای ورود دوباره کد پیامکی لازم است.', style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
                const SizedBox(height: 24),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(32)),
                  child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Container(width: 32, height: 32, decoration: BoxDecoration(color: cs.primary.withValues(alpha: 0.1), shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.info, size: 18, color: cs.primary)),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Text('در صورت مشاهده دستگاه ناشناس، بلافاصله نشست آن را قطع کرده و اگر به سیم‌کارت خود دسترسی ندارید با پشتیبانی تماس بگیرید.',
                          style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
                    ),
                  ]),
                ),
              ]),
            ),
    );
  }

  static String _platform(String p) => switch (p) {
        'android' => 'اندروید',
        'ios' => 'iOS',
        'web' => 'وب',
        _ => p,
      };

  static String _deviceName(DeviceSession s) => s.model.isNotEmpty && s.model != s.platform ? isolate(s.model) : 'دستگاه ${_platform(s.platform)}';

  Widget _card(ColorScheme cs, {required double padding, required Widget child}) => Container(
        padding: EdgeInsets.all(padding),
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(32),
          boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
        ),
        child: child,
      );

  Widget _iconDisc(ColorScheme cs, IconData icon) =>
      Container(width: 40, height: 40, decoration: BoxDecoration(color: cs.surfaceContainerHigh, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: 20, color: cs.onSurface));

  Widget _factor(ColorScheme cs, IconData icon, String title, Widget sub, Widget toggle, {String? badge}) => Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
        child: Row(children: [
          _iconDisc(cs, icon),
          const SizedBox(width: 16),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(title, style: tw(16, 22, 600, cs.onSurface)),
              if (badge != null)
                Container(
                  margin: const EdgeInsets.only(top: 4),
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(color: cs.secondaryFixed, borderRadius: BorderRadius.circular(999)),
                  child: Text(badge, style: tw(10, 12, 700, cs.onSecondaryFixed)),
                ),
              sub,
            ]),
          ),
          const SizedBox(width: 8),
          toggle,
        ]),
      );

  Widget _sectionTitle(ColorScheme cs, String title, String trailing, Color trailingColor) => Row(children: [
        Expanded(child: Text(title, style: tw(18, 24, 700, cs.onSurface))),
        Text(trailing, style: tw(12, 16, 600, trailingColor)),
      ]);

  Widget _kv(ColorScheme cs, IconData icon, String k, String v) => Row(children: [
        Sy(icon, size: 16, color: cs.onSurfaceVariant),
        const SizedBox(width: 4),
        Text(k, style: tw(12, 16, 600, cs.onSurfaceVariant)),
        const SizedBox(width: 4),
        Flexible(child: Text(v, style: tw(12, 16, 600, cs.onSurface), overflow: TextOverflow.ellipsis)),
      ]);
}
