import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/config/env.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/brand.dart';
import 'otp_screen.dart';

/// Sign-up / sign-in, step 1 of 2. Reference: archive `otp` (M-263, dark): header, hero, the authentication card
/// (phone, account type), live teaser, footer. Corrections applied per doc 19 (owner-approved in Phase 0):
/// UC-9 — the code is entered on the verification screen (6 digits); UC-5 — the implicit «با ورود … می‌پذیرید»
/// sentence becomes explicit, unticked checkboxes (terms, privacy, age) and the button stays disabled until ticked.
class WelcomeScreen extends ConsumerStatefulWidget {
  const WelcomeScreen({super.key});

  @override
  ConsumerState<WelcomeScreen> createState() => _WelcomeScreenState();
}

class _WelcomeScreenState extends ConsumerState<WelcomeScreen> {
  final _phone = TextEditingController();
  bool _seller = false;
  bool _terms = false, _privacy = false, _age = false;
  bool _busy = false;
  String? _error;

  String? get _normalized => normalizeIranMobile(_phone.text);
  bool get _ready => _normalized != null && _terms && _privacy && _age && !_busy;

  @override
  void dispose() {
    _phone.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final phone = _normalized;
    if (phone == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final session = ref.read(sessionProvider);
      final ch = await ref.read(repoProvider).requestOtp(phone, session.deviceId);
      if (!mounted) return;
      context.push('/auth/verify', extra: OtpArgs(phone: phone, challenge: ch, intendsToSell: _seller, terms: _terms, privacy: _privacy, age: _age));
    } on ApiException catch (e) {
      setState(() => _error = e.code == 'validation_failed' ? 'شماره موبایل معتبر نیست.' : e.title);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _legal(String title) => showModalBottomSheet<void>(
        context: context,
        showDragHandle: true,
        builder: (ctx) => SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              Text(title, style: tw(18, 24, 700, ctx.cs.onSurface)),
              const SizedBox(height: 8),
              Text('نسخهٔ ${toPersianDigits(Env.consentTextVersion)} — متن نهایی پس از تأیید حقوقی (D-13) منتشر می‌شود.',
                  style: tw(14, 22, 400, ctx.cs.onSurfaceVariant), textAlign: TextAlign.center),
            ]),
          ),
        ),
      );

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'ورود به ویرا',
        horizontal: 12,
        backSize: 40,
        backgroundAlpha: 0.85,
        shadow: const BoxShadow(color: Color(0x59000000), blurRadius: 12, offset: Offset(0, 1)),
        actions: [
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
          RoundIconButton(icon: Sym.share, label: 'اشتراک‌گذاری', size: 40, iconSize: 22, color: cs.onSurfaceVariant, onTap: () => soon(context, 'اشتراک‌گذاری', 'فاز ۳')),
        ],
      ),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 0, 12, 24),
          children: [
            _Hero(cs: cs),
            _card(cs),
            const SizedBox(height: 12),
            _LiveTeaser(cs: cs),
            const SizedBox(height: 16),
            Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(Sym.verifiedUser, size: 16, color: cs.onSurfaceVariant.withValues(alpha: 0.6)),
              const SizedBox(width: 4),
              Text('ارتباط امن رمزنگاری‌شده سرتاسری', style: tw(11, 14, 500, cs.onSurfaceVariant.withValues(alpha: 0.6))),
            ]),
          ],
        ),
      ),
    );
  }

  Widget _card(ColorScheme cs) {
    final valid = _normalized != null;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surfaceContainer,
        borderRadius: BorderRadius.circular(12),
        boxShadow: const [
          BoxShadow(color: Color(0x1A000000), blurRadius: 25, offset: Offset(0, 20), spreadRadius: -5),
          BoxShadow(color: Color(0x1A000000), blurRadius: 10, offset: Offset(0, 8), spreadRadius: -6),
        ],
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Row(children: [
          Container(
            width: 10,
            height: 10,
            decoration: BoxDecoration(color: cs.primaryContainer, shape: BoxShape.circle, boxShadow: [BoxShadow(color: cs.primaryContainer.withValues(alpha: 0.5), blurRadius: 8)]),
          ),
          const SizedBox(width: 4),
          Text('احراز هویت پیامکی', style: tw(12, 16, 600, cs.primaryFixed)),
          const Spacer(),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(color: cs.surfaceContainerHigh, borderRadius: BorderRadius.circular(999)),
            child: Text('مرحله ۱ از ۲', style: tw(11, 14, 500, cs.onSurfaceVariant)),
          ),
        ]),
        const SizedBox(height: 16),
        Row(children: [
          Text('شماره تلفن همراه', style: tw(12, 16, 600, cs.onSurface)),
          const Spacer(),
          Text('ارسال کد امنیتی', style: tw(11, 14, 500, cs.onSurfaceVariant)),
        ]),
        const SizedBox(height: 4),
        ClipRRect(
          borderRadius: BorderRadius.circular(8),
          child: Container(
            height: 48,
            color: cs.surfaceContainerHigh,
            child: Row(children: [
              Container(
                width: 70,
                color: cs.surfaceContainerHighest,
                alignment: Alignment.center,
                child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                  const Text('🇮🇷', style: TextStyle(fontSize: 16, height: 1)),
                  const SizedBox(width: 6),
                  Text('\u2066+۹۸\u2069', style: tw(12, 16, 600, cs.primaryFixed)),
                ]),
              ),
              Expanded(
                child: TextField(
                  controller: _phone,
                  key: const Key('phone'),
                  textDirection: TextDirection.ltr,
                  textAlign: TextAlign.left,
                  keyboardType: TextInputType.phone,
                  autofillHints: const [AutofillHints.telephoneNumber],
                  inputFormatters: [FilteringTextInputFormatter.allow(RegExp(r'[0-9۰-۹٠-٩+ ]')), LengthLimitingTextInputFormatter(14)],
                  onChanged: (_) => setState(() => _error = null),
                  style: tw(16, 24, 400, cs.onSurface, spacing: 1.6),
                  cursorColor: cs.primaryContainer,
                  decoration: InputDecoration(
                    isCollapsed: true,
                    border: InputBorder.none,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                    hintText: '۹۱۲ ۳۴۵ ۶۷۸۹',
                    hintTextDirection: TextDirection.ltr,
                    hintStyle: tw(16, 24, 400, cs.onSurfaceVariant.withValues(alpha: 0.5), spacing: 1.6),
                  ),
                ),
              ),
              SizedBox(
                width: 44,
                child: valid ? Sy(Sym.checkCircle, size: 20, color: cs.primaryContainer, fill: true) : null,
              ),
            ]),
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 6),
          Text(_error!, style: tw(11, 14, 500, cs.error), key: const Key('welcome-error')),
        ],
        const SizedBox(height: 16),
        Text('نوع کاربری خود را انتخاب کنید', style: tw(12, 16, 600, cs.onSurface)),
        const SizedBox(height: 4),
        Row(children: [
          Expanded(
            child: _TypeCard(
              selected: !_seller,
              icon: Sym.favorite,
              iconColor: cs.primaryContainer,
              title: 'کاربر سوشال',
              subtitle: 'خرید و تعامل با سازندگان',
              onTap: () => setState(() => _seller = false),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _TypeCard(
              selected: _seller,
              icon: Sym.storefront,
              iconColor: cs.secondary,
              title: 'فروشنده رسمی',
              subtitle: 'افتتاح فروشگاه ویرا',
              onTap: () => setState(() => _seller = true),
            ),
          ),
        ]),
        const SizedBox(height: 12),
        // UC-5: one unticked box per purpose; each label opens the text (version recorded with the consent).
        ArchiveCheckbox(
          key: const Key('consent-terms'),
          value: _terms,
          onChanged: (v) => setState(() => _terms = v),
          child: _consentText(cs, 'شرایط استفاده', 'ویرا را خوانده‌ام و می‌پذیرم.', () => _legal('شرایط استفاده')),
        ),
        ArchiveCheckbox(
          key: const Key('consent-privacy'),
          value: _privacy,
          onChanged: (v) => setState(() => _privacy = v),
          child: _consentText(cs, 'سیاست حفظ حریم خصوصی', 'را خوانده‌ام و می‌پذیرم.', () => _legal('سیاست حفظ حریم خصوصی')),
        ),
        ArchiveCheckbox(
          key: const Key('consent-age'),
          value: _age,
          onChanged: (v) => setState(() => _age = v),
          child: Text('تأیید می‌کنم که دست‌کم ${toPersianDigits(Env.minimumAge)} سال سن دارم.', style: tw(12, 18, 500, cs.onSurfaceVariant)),
        ),
        const SizedBox(height: 12),
        _GradientButton(
          key: const Key('welcome-submit'),
          label: _busy ? 'در حال ارسال کد…' : 'تایید و ورود به ویرا',
          icon: Sym.login,
          enabled: _ready,
          onTap: _submit,
        ),
      ]),
    );
  }

  Widget _consentText(ColorScheme cs, String link, String rest, VoidCallback open) => Text.rich(
        TextSpan(children: [
          WidgetSpan(
            alignment: PlaceholderAlignment.baseline,
            baseline: TextBaseline.alphabetic,
            child: GestureDetector(onTap: open, child: Text(link, style: tw(11, 18, 500, cs.primary))),
          ),
          TextSpan(text: ' $rest', style: tw(12, 18, 400, cs.onSurfaceVariant)),
        ]),
      );
}

class _Hero extends StatelessWidget {
  const _Hero({required this.cs});
  final ColorScheme cs;

  @override
  Widget build(BuildContext context) => SizedBox(
        height: 188,
        child: Stack(clipBehavior: Clip.none, alignment: Alignment.topCenter, children: [
          Positioned(
            top: -40,
            child: Container(
              width: 192,
              height: 192,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: cs.primaryContainer.withValues(alpha: 0.2),
                boxShadow: [BoxShadow(color: cs.primaryContainer.withValues(alpha: 0.2), blurRadius: 64, spreadRadius: 16)],
              ),
            ),
          ),
          Positioned(
            top: 12,
            child: Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                color: cs.surfaceContainerHigh,
                borderRadius: BorderRadius.circular(12),
                boxShadow: const [BoxShadow(color: Color(0x1A000000), blurRadius: 25, offset: Offset(0, 20), spreadRadius: -5)],
              ),
              child: Stack(children: [
                Positioned.fill(
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(12),
                      gradient: LinearGradient(
                        begin: Alignment.bottomLeft,
                        end: Alignment.topRight,
                        colors: [cs.primaryContainer.withValues(alpha: 0.3), cs.secondary.withValues(alpha: 0.1)],
                      ),
                    ),
                  ),
                ),
                Center(child: ViraMark(size: 48, color: cs.primaryContainer)),
                Positioned(
                  top: 6,
                  right: 6,
                  child: Container(width: 10, height: 10, decoration: BoxDecoration(color: cs.primaryContainer, shape: BoxShape.circle)),
                ),
              ]),
            ),
          ),
          Positioned(
            top: 104,
            left: 0,
            right: 0,
            child: Column(children: [
              Text.rich(
                TextSpan(style: tw(20, 28, 700, cs.onSurface), children: [
                  const TextSpan(text: 'به بازارگاه و سوشال‌کامرس '),
                  TextSpan(text: 'ویرا', style: TextStyle(color: cs.primaryContainer)),
                  const TextSpan(text: ' خوش آمدید'),
                ]),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 4),
              ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 280),
                child: Text('تجربه خریدی لذت‌بخش و تعاملی در تماشایی‌ترین پلتفرم ویدیویی خاورمیانه',
                    style: tw(12, 18, 400, cs.onSurfaceVariant), textAlign: TextAlign.center),
              ),
            ]),
          ),
        ]),
      );
}

class _TypeCard extends StatelessWidget {
  const _TypeCard({required this.selected, required this.icon, required this.iconColor, required this.title, required this.subtitle, required this.onTap});
  final bool selected;
  final IconData icon;
  final Color iconColor;
  final String title, subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Semantics(
      selected: selected,
      button: true,
      label: title,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          height: 84,
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: Color.alphaBlend(selected ? cs.primaryContainer.withValues(alpha: 0.1) : Colors.transparent, cs.surfaceContainerHigh),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(color: cs.surfaceContainer, shape: BoxShape.circle),
                alignment: Alignment.center,
                child: Sy(icon, size: 18, color: iconColor),
              ),
              const Spacer(),
              Container(
                width: 16,
                height: 16,
                decoration: BoxDecoration(color: selected ? cs.primaryContainer : cs.surfaceContainerHighest, shape: BoxShape.circle),
                alignment: Alignment.center,
                child: selected ? const Sy(Sym.check, size: 12, color: Colors.white) : null,
              ),
            ]),
            const SizedBox(height: 4),
            Text(title, style: tw(12, 16, 600, cs.onSurface)),
            const SizedBox(height: 2),
            Text(subtitle, style: tw(11, 14, 500, cs.onSurfaceVariant), maxLines: 1, overflow: TextOverflow.ellipsis),
          ]),
        ),
      ),
    );
  }
}

/// `h-12 rounded-lg bg-gradient-to-r from-primary-container to-inverse-primary` primary action of the welcome card.
class _GradientButton extends StatelessWidget {
  const _GradientButton({super.key, required this.label, required this.icon, required this.enabled, required this.onTap});
  final String label;
  final IconData icon;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Semantics(
      button: true,
      enabled: enabled,
      label: label,
      child: Opacity(
        opacity: enabled ? 1 : 0.45,
        child: GestureDetector(
          onTap: enabled ? onTap : null,
          child: Container(
            height: 48,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(8),
              gradient: LinearGradient(colors: [cs.primaryContainer, cs.inversePrimary], begin: AlignmentDirectional.centerEnd, end: AlignmentDirectional.centerStart),
              boxShadow: [BoxShadow(color: cs.primaryContainer.withValues(alpha: 0.35), blurRadius: 20, offset: const Offset(0, 4))],
            ),
            child: ExcludeSemantics(
              child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Sy(icon, size: 20, color: Colors.white),
                const SizedBox(width: 8),
                Text(label, style: tw(14, 20, 600, Colors.white)),
              ]),
            ),
          ),
        ),
      ),
    );
  }
}

/// Live-shopping teaser kept from the reference; live auctions are Phase 8, so it shows that instead of sample data.
class _LiveTeaser extends StatelessWidget {
  const _LiveTeaser({required this.cs});
  final ColorScheme cs;

  @override
  Widget build(BuildContext context) => GestureDetector(
        onTap: () => soon(context, 'حراج و خرید زنده', 'فاز ۸'),
        child: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: cs.surfaceContainerLow,
            borderRadius: BorderRadius.circular(12),
            boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
          ),
          child: Row(children: [
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: SizedBox(
                width: 48,
                height: 48,
                child: Stack(children: [
                  Positioned.fill(child: ColoredBox(color: cs.surfaceContainerHigh, child: Center(child: Sy(Sym.liveTv, size: 22, color: cs.onSurfaceVariant)))),
                  Positioned(
                    left: 0,
                    right: 0,
                    bottom: 0,
                    height: 16,
                    child: ColoredBox(
                      color: cs.primaryContainer.withValues(alpha: 0.9),
                      child: Center(child: Text('لایو', style: tw(9, 9, 700, Colors.white))),
                    ),
                  ),
                ]),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Flexible(child: Text('حراج‌های زنده ویرا', style: tw(12, 16, 600, cs.onSurface), overflow: TextOverflow.ellipsis)),
                  const SizedBox(width: 6),
                  Container(width: 6, height: 6, decoration: BoxDecoration(color: cs.primaryContainer, shape: BoxShape.circle)),
                ]),
                const SizedBox(height: 2),
                Text('پخش زنده و خرید آنلاین به‌زودی در ویرا', style: tw(12, 18, 400, cs.onSurfaceVariant), maxLines: 1, overflow: TextOverflow.ellipsis),
              ]),
            ),
            const SizedBox(width: 12),
            Sy(Sym.arrowBack, size: 22, color: cs.primary),
          ]),
        ),
      );
}
