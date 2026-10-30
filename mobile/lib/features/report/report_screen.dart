import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';

class ReportArgs {
  const ReportArgs({required this.targetType, required this.targetId, this.title, this.subtitle});
  final String targetType, targetId;
  final String? title, subtitle;
}

/// Opens the report form for a post, comment or account.
void showReportSheet(BuildContext context, {required String targetType, required String targetId, String? title, String? subtitle}) =>
    context.push('/report', extra: ReportArgs(targetType: targetType, targetId: targetId, title: title, subtitle: subtitle));

/// Report a violation. Reference: `_213` (M-213, light; dark `_210`): confidentiality card, reported item, reason
/// radio cards, details + attachments, block-at-the-same-time switch, review-process card, sticky send button.
/// The reference's five commerce reasons map to the API reasons; the social-safety reasons the API also
/// accepts (hate, violence, nudity, self-harm, misinformation) are added as further cards of the same design
/// (OD-P2-10). Attachments are a later phase. Wording corrections: «۱۰۰٪ ناشناس» and «رمزنگاری سرتاسری»
/// overstated what Phase 2 does — see docs/progress/P2.md §Security UI.
class ReportScreen extends ConsumerStatefulWidget {
  const ReportScreen({super.key, required this.args});
  final ReportArgs args;

  @override
  ConsumerState<ReportScreen> createState() => _ReportScreenState();
}

class _ReportScreenState extends ConsumerState<ReportScreen> {
  static const reasons = [
    ('scam', Sym.gavel, 'کلاهبرداری، کالای تقلبی و غیرمجاز', 'فروش کالای فیک، عدم ارسال، درگاه پرداخت جعلی یا اقلام ممنوعه'),
    ('counterfeit', Sym.copyright, 'نقض مالکیت فکری یا کپی‌رایت', 'سوءاستفاده از لوگو، برند، عکس محصولات دیگران بدون مجوز رسمی'),
    ('harassment', Sym.warning, 'آزار، توهین یا محتوای نامناسب', 'توهین به مقدسات یا اقوام، تهدید یا آزار کاربران'),
    ('spam', Sym.sell, 'قیمت‌گذاری فریبنده یا اسپم', 'پیام‌های تبلیغاتی مکرر یا قیمت‌گذاری گمراه‌کننده'),
    ('hate', Sym.block, 'نفرت‌پراکنی', 'تحقیر یا حمله به افراد به خاطر قومیت، دین، جنسیت یا ویژگی‌های دیگر'),
    ('violence', Sym.localFireDepartment, 'خشونت یا تهدید', 'تصاویر خشن، تهدید به آسیب یا ترویج خشونت'),
    ('nudity', Sym.visibilityOff, 'برهنگی یا محتوای جنسی', 'تصاویر یا متن غیراخلاقی و نامناسب'),
    ('self_harm', Sym.shieldWithHeart, 'آسیب به خود', 'محتوایی که به آسیب‌رساندن به خود اشاره یا تشویق می‌کند'),
    ('misinformation', Sym.info, 'اطلاعات نادرست', 'ادعاهای گمراه‌کننده یا اخبار جعلی'),
    ('other', Sym.receiptLong, 'نقض قوانین تجارت اجتماعی ویرا', 'سایر موارد نقض قوانین و شرایط استفاده'),
  ];

  String? _reason;
  final _details = TextEditingController();
  bool _block = false, _busy = false;

  @override
  void dispose() {
    _details.dispose();
    super.dispose();
  }

  String get _typeLabel => switch (widget.args.targetType) {
        'post' => 'پست',
        'comment' => 'نظر',
        _ => 'حساب',
      };

  Future<void> _send() async {
    final reason = _reason;
    if (reason == null || _busy) return;
    setState(() => _busy = true);
    final repo = ref.read(repoProvider);
    try {
      await repo.report(targetType: widget.args.targetType, targetId: widget.args.targetId, reason: reason, details: _details.text.trim());
      if (_block && widget.args.targetType == 'user') await repo.block(widget.args.targetId);
      if (!mounted) return;
      toast(context, 'گزارش شما ثبت شد. سپاس از همراهی شما.');
      context.pop(true);
    } on ApiException catch (e) {
      if (mounted) toast(context, e.code == 'conflict' ? 'این مورد را قبلاً گزارش کرده‌اید و در حال بررسی است.' : e.title);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final a = widget.args;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(title: 'گزارش تخلف', height: 64, horizontal: 16, backgroundAlpha: 0.8, titleStyle: tw(18, 24, 700, cs.onSurface), actions: const [HeaderPersonChip()]),
      bottomNavigationBar: Container(
        color: cs.surface.withValues(alpha: 0.9),
        padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
        child: SafeArea(
          top: false,
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Opacity(
              opacity: _reason == null ? 0.5 : 1,
              child: Material(
                color: cs.primaryContainer,
                borderRadius: BorderRadius.circular(999),
                elevation: 0,
                child: InkWell(
                  key: const Key('report-send'),
                  borderRadius: BorderRadius.circular(999),
                  onTap: _reason == null ? null : _send,
                  child: SizedBox(
                    height: 56,
                    child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                      Sy(Sym.send, size: 22, color: cs.onPrimaryContainer),
                      const SizedBox(width: 8),
                      Text(_busy ? 'در حال ارسال…' : 'ارسال محرمانه گزارش تخلف', style: tw(14, 18, 600, cs.onPrimaryContainer)),
                    ]),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 8),
            Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(Sym.enhancedEncryption, size: 16, color: cs.onSurfaceVariant),
              const SizedBox(width: 4),
              Text('ارسال رمزنگاری‌شده و حفاظت از حریم شخصی گزارش‌دهنده', style: tw(10, 12, 700, cs.onSurfaceVariant)),
            ]),
          ]),
        ),
      ),
      body: ListView(padding: const EdgeInsets.fromLTRB(16, 16, 16, 24), children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
          child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Row(children: [
              Container(width: 40, height: 40, decoration: BoxDecoration(color: cs.secondaryContainer.withValues(alpha: 0.1), shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.verifiedUser, size: 24, color: cs.secondary, fill: true)),
              const SizedBox(width: 8),
              Expanded(child: Text('گزارش محرمانه و امن', style: tw(16, 22, 600, cs.onSurface))),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(color: cs.secondary.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(999)),
                child: Row(mainAxisSize: MainAxisSize.min, children: [
                  Text('محرمانه', style: tw(10, 12, 700, cs.secondary)),
                  const SizedBox(width: 2),
                  Sy(Sym.lock, size: 14, color: cs.secondary),
                ]),
              ),
            ]),
            const SizedBox(height: 8),
            Text('هویت، نام کاربری و شماره تماس شما به هیچ عنوان به حساب گزارش‌شده اعلام نخواهد شد. گزارش فقط برای تیم بررسی ویرا قابل مشاهده است.',
                style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
          ]),
        ),
        const SizedBox(height: 24),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
          child: Row(children: [
            Sy(Sym.flag, size: 16, color: cs.onSurfaceVariant),
            const SizedBox(width: 6),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(a.title ?? '$_typeLabel گزارش‌شده', style: tw(16, 22, 600, cs.onSurface)),
                if (a.subtitle != null) Text(a.subtitle!, style: tw(12, 16, 600, cs.onSurfaceVariant)),
                Text('شناسه: ${isolate(a.targetId.length > 8 ? a.targetId.substring(0, 8) : a.targetId)}', style: tw(12, 16, 600, cs.onSurfaceVariant)),
              ]),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(color: cs.primaryFixed, borderRadius: BorderRadius.circular(999)),
              child: Text(_typeLabel, style: tw(10, 12, 700, cs.onPrimaryFixedVariant)),
            ),
          ]),
        ),
        const SizedBox(height: 24),
        _heading(cs, 'علت تخلف را انتخاب کنید', 'الزامی', cs.primaryFixed, cs.onPrimaryFixedVariant),
        const SizedBox(height: 8),
        for (final (code, icon, title, sub) in reasons) ...[
          _reasonCard(cs, code, icon, title, sub),
          const SizedBox(height: 4),
        ],
        const SizedBox(height: 20),
        _heading(cs, 'شرح و مدارک تکمیلی', 'اختیاری', cs.surfaceContainerHigh, cs.onSurfaceVariant),
        const SizedBox(height: 8),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
          child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
              child: TextField(
                key: const Key('report-details'),
                controller: _details,
                minLines: 4,
                maxLines: 6,
                inputFormatters: [LengthLimitingTextInputFormatter(500)],
                onChanged: (_) => setState(() {}),
                style: tw(14, 22, 400, cs.onSurface),
                decoration: InputDecoration(
                  isCollapsed: true,
                  border: InputBorder.none,
                  hintText: 'توضیح کوتاهی دربارهٔ تخلف بنویسید…',
                  hintStyle: tw(14, 22, 400, cs.onSurfaceVariant.withValues(alpha: 0.6)),
                ),
              ),
            ),
            const SizedBox(height: 4),
            Align(alignment: AlignmentDirectional.centerEnd, child: Text('${toPersianDigits(_details.text.length)} / ۵۰۰ حرف', style: tw(10, 12, 700, cs.onSurfaceVariant))),
            const SizedBox(height: 16),
            Text('بارگذاری اسناد، فیش یا اسکرین‌شات (حداکثر ۳ فایل)', style: tw(12, 16, 600, cs.onSurface)),
            const SizedBox(height: 4),
            Row(children: [
              Expanded(
                child: GestureDetector(
                  onTap: () => soon(context, 'پیوست گزارش', 'نسخه‌های بعد'),
                  child: Container(
                    height: 96,
                    decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(48)),
                    child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                      Container(width: 32, height: 32, decoration: BoxDecoration(color: cs.primaryFixed, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.addAPhoto, size: 20, color: cs.onPrimaryFixedVariant)),
                      const SizedBox(height: 4),
                      Text('افزودن فایل', style: tw(10, 12, 600, cs.primaryContainer)),
                    ]),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              for (final i in const [2, 3]) ...[
                Expanded(
                  child: Container(
                    height: 96,
                    decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(48)),
                    child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                      Sy(Sym.image, size: 24, color: cs.onSurfaceVariant.withValues(alpha: 0.5)),
                      const SizedBox(height: 4),
                      Text('جایگاه ${toPersianDigits(i)}', style: tw(10, 12, 700, cs.onSurfaceVariant.withValues(alpha: 0.7))),
                    ]),
                  ),
                ),
                if (i == 2) const SizedBox(width: 8),
              ],
            ]),
          ]),
        ),
        if (a.targetType == 'user') ...[
          const SizedBox(height: 24),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(color: cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(16), boxShadow: _sm),
            child: Row(children: [
              Container(width: 40, height: 40, decoration: BoxDecoration(color: cs.errorContainer.withValues(alpha: 0.4), shape: BoxShape.circle), alignment: Alignment.center, child: Sy(Sym.block, size: 22, color: cs.error)),
              const SizedBox(width: 16),
              Expanded(
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text('مسدودسازی همزمان این کاربر', style: tw(16, 22, 600, cs.onSurface)),
                  Text('دیگر قادر به مشاهده پست‌ها و پیام به شما نخواهد بود', style: tw(14, 20, 400, cs.onSurfaceVariant)),
                ]),
              ),
              const SizedBox(width: 8),
              ArchiveSwitch(key: const Key('report-block'), value: _block, label: 'مسدودسازی همزمان', onChanged: (v) => setState(() => _block = v)),
            ]),
          ),
        ],
        const SizedBox(height: 24),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: cs.surfaceContainerLow, borderRadius: BorderRadius.circular(16)),
          child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Sy(Sym.neurology, size: 22, color: cs.secondary),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('فرآیند بررسی کارشناسی ویرا', style: tw(16, 22, 600, cs.onSurface)),
                const SizedBox(height: 2),
                Text('گزارش شما در صف بررسی تیم نظارت محتوای ویرا قرار می‌گیرد و پس از بررسی، اقدام لازم (حذف محتوا یا محدودسازی حساب) انجام می‌شود.',
                    style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
              ]),
            ),
          ]),
        ),
      ]),
    );
  }

  static const _sm = [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))];

  Widget _heading(ColorScheme cs, String title, String badge, Color bg, Color fg) => Row(children: [
        Expanded(child: Text(title, style: tw(18, 24, 700, cs.onSurface))),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
          decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(999)),
          child: Text(badge, style: tw(10, 12, 700, fg)),
        ),
      ]);

  Widget _reasonCard(ColorScheme cs, String code, IconData icon, String title, String sub) {
    final sel = _reason == code;
    return Semantics(
      selected: sel,
      button: true,
      child: GestureDetector(
        key: Key('reason-$code'),
        onTap: () => setState(() => _reason = code),
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(color: sel ? cs.primaryFixed.withValues(alpha: 0.2) : cs.surfaceContainerLowest, borderRadius: BorderRadius.circular(16)),
          child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Container(
              margin: const EdgeInsets.only(top: 2),
              width: 24,
              height: 24,
              decoration: BoxDecoration(color: sel ? cs.primaryContainer : cs.surfaceContainer, shape: BoxShape.circle),
              alignment: Alignment.center,
              child: sel ? Container(width: 10, height: 10, decoration: BoxDecoration(color: cs.onPrimaryContainer, shape: BoxShape.circle)) : null,
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Expanded(child: Text(title, style: tw(16, 22, 600, cs.onSurface))),
                  Sy(icon, size: 20, color: sel ? cs.primaryContainer : cs.onSurfaceVariant),
                ]),
                const SizedBox(height: 2),
                Text(sub, style: tw(14, 22.75, 400, cs.onSurfaceVariant)),
              ]),
            ),
          ]),
        ),
      ),
    );
  }
}
