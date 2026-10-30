import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/config/env.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';

class OtpArgs {
  const OtpArgs({required this.phone, required this.challenge, required this.intendsToSell, required this.terms, required this.privacy, required this.age});
  final String phone;
  final OtpChallenge challenge;
  final bool intendsToSell, terms, privacy, age;
}

/// Sign-in step 2: one-time code. Reference: archive `mobile_otp_verification_light` (M-258) — icon group, masked
/// number, edit-number link, digit boxes, resend countdown pill, confidentiality card, primary button.
/// UC-9: 6 digits (the reference draws 5), SMS autofill hint, lockout message after 5 wrong attempts.
class OtpScreen extends ConsumerStatefulWidget {
  const OtpScreen({super.key, required this.args});
  final OtpArgs args;

  @override
  ConsumerState<OtpScreen> createState() => _OtpScreenState();
}

class _OtpScreenState extends ConsumerState<OtpScreen> {
  final _code = TextEditingController();
  final _focus = FocusNode();
  late OtpChallenge _challenge = widget.args.challenge;
  late int _resendIn = _challenge.resendAfter;
  Timer? _timer;
  bool _busy = false, _locked = false;
  String? _error;

  int get _len => _challenge.codeLength;

  @override
  void initState() {
    super.initState();
    _startTimer();
  }

  void _startTimer() {
    _timer?.cancel();
    _timer = Timer.periodic(const Duration(seconds: 1), (t) {
      if (_resendIn <= 0) {
        t.cancel();
        return;
      }
      setState(() => _resendIn--);
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    _code.dispose();
    _focus.dispose();
    super.dispose();
  }

  Future<void> _verify() async {
    final code = toLatinDigits(_code.text);
    if (code.length != _len || _busy || _locked) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final session = ref.read(sessionProvider);
    try {
      final res = await ref.read(repoProvider).verifyOtp(
            challengeId: _challenge.id,
            code: code,
            device: session.deviceInfo,
            intendsToSell: widget.args.intendsToSell,
            terms: widget.args.terms,
            privacy: widget.args.privacy,
            ageConfirmation: widget.args.age,
            textVersion: Env.consentTextVersion,
          );
      await session.accept(res);
      if (!mounted) return;
      context.go(res['is_new'] == true ? '/profile/edit' : '/home');
    } on ApiException catch (e) {
      setState(() {
        _code.clear();
        _locked = e.code == 'otp_locked';
        _error = switch (e.code) {
          'otp_invalid' => 'کد واردشده درست نیست.',
          'otp_expired' => 'کد منقضی شده است. کد جدید دریافت کنید.',
          'otp_locked' => 'به دلیل تلاش‌های ناموفق، این کد قفل شد. کد جدید دریافت کنید.',
          _ => e.title,
        };
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _resend() async {
    if (_resendIn > 0 || _busy) return;
    setState(() => _busy = true);
    try {
      final ch = await ref.read(repoProvider).requestOtp(widget.args.phone, ref.read(sessionProvider).deviceId);
      setState(() {
        _challenge = ch;
        _resendIn = ch.resendAfter;
        _locked = false;
        _error = null;
        _code.clear();
      });
      _startTimer();
      if (mounted) toast(context, 'کد جدید ارسال شد.');
    } on ApiException catch (e) {
      setState(() => _error = e.title);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String get _masked {
    final p = widget.args.phone;
    return '\u2066${toPersianDigits('${p.substring(0, 4)}***${p.substring(p.length - 4)}')}\u2069';
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final mm = toPersianDigits('${(_resendIn ~/ 60).toString().padLeft(2, '0')}:${(_resendIn % 60).toString().padLeft(2, '0')}');
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: _Header(cs: cs),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
          children: [
            Center(
              child: SizedBox(
                width: 88,
                height: 84,
                child: Stack(clipBehavior: Clip.none, children: [
                  Positioned(
                    left: 4,
                    top: 0,
                    child: Container(
                      width: 80,
                      height: 80,
                      decoration: BoxDecoration(
                        color: cs.surfaceContainer,
                        shape: BoxShape.circle,
                        boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
                      ),
                      alignment: Alignment.center,
                      child: Container(
                        width: 56,
                        height: 56,
                        decoration: BoxDecoration(color: cs.secondaryFixed.withValues(alpha: 0.4), shape: BoxShape.circle),
                        alignment: Alignment.center,
                        child: Sy(Sym.verifiedUser, size: 32, color: cs.secondary, fill: true),
                      ),
                    ),
                  ),
                  Positioned(
                    right: 0,
                    top: 56,
                    child: Container(
                      width: 28,
                      height: 28,
                      decoration: BoxDecoration(
                        color: cs.primary,
                        shape: BoxShape.circle,
                        boxShadow: const [
                          BoxShadow(color: Color(0x1A000000), blurRadius: 6, offset: Offset(0, 4), spreadRadius: -1),
                          BoxShadow(color: Color(0x1A000000), blurRadius: 4, offset: Offset(0, 2), spreadRadius: -2),
                        ],
                      ),
                      alignment: Alignment.center,
                      child: Sy(Sym.sms, size: 16, color: cs.onPrimary),
                    ),
                  ),
                ]),
              ),
            ),
            const SizedBox(height: 12),
            Text('کد تأیید را وارد کنید', style: tw(22, 28, 700, cs.onSurface), textAlign: TextAlign.center),
            const SizedBox(height: 4),
            Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 280),
                child: Text.rich(
                  TextSpan(style: tw(14, 22.75, 400, cs.onSurfaceVariant), children: [
                    TextSpan(text: 'کد ${toPersianDigits(_len)} رقمی ارسال‌شده به شماره '),
                    TextSpan(text: _masked, style: tw(14, 22.75, 700, cs.onSurface, spacing: 0.7)),
                    const TextSpan(text: ' را وارد نمایید.'),
                  ]),
                  textAlign: TextAlign.center,
                ),
              ),
            ),
            const SizedBox(height: 4),
            Center(
              child: InkWell(
                onTap: () => context.pop(),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Sy(Sym.edit, size: 16, color: cs.secondary),
                  const SizedBox(width: 2),
                  Text('ویرایش شماره', style: tw(12, 16, 600, cs.secondary)),
                ]),
              ),
            ),
            const SizedBox(height: 24),
            _boxes(cs),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, key: const Key('otp-error'), style: tw(12, 18, 600, cs.error), textAlign: TextAlign.center),
            ],
            const SizedBox(height: 24),
            Center(
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
                decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(999)),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Sy(Sym.schedule, size: 18, color: cs.primary),
                  const SizedBox(width: 4),
                  Text.rich(TextSpan(children: [
                    TextSpan(text: '\u2066$mm\u2069', style: tw(12, 16, 700, cs.onSurface, spacing: 0.6)),
                    TextSpan(text: ' تا امکان ارسال مجدد کد', style: tw(12, 16, 600, cs.onSurfaceVariant)),
                  ])),
                ]),
              ),
            ),
            const SizedBox(height: 4),
            Center(
              child: InkWell(
                key: const Key('otp-resend'),
                onTap: _resendIn > 0 ? null : _resend,
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    Sy(Sym.refresh, size: 16, color: _resendIn > 0 ? cs.onSurfaceVariant.withValues(alpha: 0.5) : cs.primary),
                    const SizedBox(width: 2),
                    Text('ارسال مجدد از طریق پیامک', style: tw(12, 16, 600, _resendIn > 0 ? cs.onSurfaceVariant.withValues(alpha: 0.5) : cs.primary)),
                  ]),
                ),
              ),
            ),
            const SizedBox(height: 24),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
              child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Container(
                  margin: const EdgeInsets.only(top: 2),
                  width: 32,
                  height: 32,
                  decoration: BoxDecoration(color: cs.surfaceContainerHighest, shape: BoxShape.circle),
                  alignment: Alignment.center,
                  child: Sy(Sym.gppMaybe, size: 18, color: cs.secondary),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Text('محرمانگی کد تأیید', style: tw(12, 16, 700, cs.onSurface)),
                    const SizedBox(height: 2),
                    Text('کد یک‌بار مصرف صرفاً جهت تأیید هویت کاربری بوده و هرگز آن را در اختیار افراد دیگر قرار ندهید.',
                        style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
                  ]),
                ),
              ]),
            ),
            const SizedBox(height: 32),
            Semantics(
              button: true,
              enabled: _code.text.length == _len && !_busy && !_locked,
              child: Material(
                color: cs.primary,
                borderRadius: BorderRadius.circular(16),
                elevation: 0,
                shadowColor: Colors.black26,
                child: InkWell(
                  key: const Key('otp-submit'),
                  borderRadius: BorderRadius.circular(16),
                  onTap: _verify,
                  child: SizedBox(
                    height: 52,
                    child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                      Text(_busy ? 'در حال بررسی…' : 'تأیید و ادامه', style: tw(14, 18, 700, cs.onPrimary)),
                      const SizedBox(width: 4),
                      Sy(Sym.arrowBack, size: 20, color: cs.onPrimary),
                    ]),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 8),
            Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(Sym.lock, size: 14, color: cs.tertiary),
              const SizedBox(width: 4),
              Text('رمزنگاری امن دوطرفه سامانه ویرا', style: tw(10, 12, 700, cs.onSurfaceVariant)),
            ]),
          ],
        ),
      ),
    );
  }

  /// Digit boxes (`w-12 h-14 rounded-xl bg-surface-container-lowest shadow-sm`, current box `shadow-md` + caret).
  /// A transparent field on top receives the input, so paste and SMS autofill (oneTimeCode) work.
  Widget _boxes(ColorScheme cs) {
    final digits = toPersianDigits(toLatinDigits(_code.text));
    return Center(
      child: SizedBox(
        width: 320,
        height: 56,
        child: Stack(children: [
          Directionality(
            textDirection: TextDirection.ltr,
            child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              for (var i = 0; i < _len; i++) ...[
                if (i > 0) const SizedBox(width: 10),
                Container(
                  width: (320 - 10 * (_len - 1)) / _len,
                  height: 56,
                  decoration: BoxDecoration(
                    color: cs.surfaceContainerLowest,
                    borderRadius: BorderRadius.circular(48),
                    boxShadow: i == digits.length
                        ? const [
                            BoxShadow(color: Color(0x1A000000), blurRadius: 6, offset: Offset(0, 4), spreadRadius: -1),
                            BoxShadow(color: Color(0x1A000000), blurRadius: 4, offset: Offset(0, 2), spreadRadius: -2),
                          ]
                        : const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
                  ),
                  alignment: Alignment.center,
                  child: i < digits.length
                      ? Text(digits[i], style: tw(28, 34, 700, cs.onSurface))
                      : (i == digits.length && !_locked
                          ? Container(width: 2, height: 24, decoration: BoxDecoration(color: cs.primary, borderRadius: BorderRadius.circular(2)))
                          : null),
                ),
              ],
            ]),
          ),
          Positioned.fill(
            child: Opacity(
              opacity: 0,
              child: TextField(
                key: const Key('otp-input'),
                controller: _code,
                focusNode: _focus,
                autofocus: true,
                enabled: !_locked,
                keyboardType: TextInputType.number,
                autofillHints: const [AutofillHints.oneTimeCode],
                showCursor: false,
                enableInteractiveSelection: false,
                inputFormatters: [FilteringTextInputFormatter.allow(RegExp(r'[0-9۰-۹٠-٩]')), LengthLimitingTextInputFormatter(_len)],
                onChanged: (v) {
                  setState(() => _error = null);
                  if (v.length == _len) _verify();
                },
                decoration: const InputDecoration(border: InputBorder.none, counterText: ''),
              ),
            ),
          ),
        ]),
      ),
    );
  }
}

class _Header extends StatelessWidget implements PreferredSizeWidget {
  const _Header({required this.cs});
  final ColorScheme cs;

  @override
  Size get preferredSize => const Size.fromHeight(64);

  @override
  Widget build(BuildContext context) => ArchiveHeader(
        title: 'تأیید شماره موبایل',
        height: 64,
        horizontal: 16,
        centerTitle: true,
        backgroundAlpha: 0.8,
        titleStyle: tw(18, 24, 700, cs.onSurface),
        actions: const [HeaderPersonChip()],
        backIcon: Sym.arrowBack, // the reference rotates arrow_forward by 180°
        onBack: () => context.pop(),
      );
}
