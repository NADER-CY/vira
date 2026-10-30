import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/api/repository.dart';
import '../../core/format/persian.dart';
import '../../core/media/picker.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';

/// Create / edit a post. Reference: `create_post_light` (M-243, "Creator Studio"): top card (close, title,
/// preview, publish), media section (cover + slides + add), caption with counter, linked products, publishing
/// options, publish and draft buttons. Phase 2 posts are images + caption + audience; product tags (P4),
/// story cross-post (P3), one-click buy (P4), location and drafts keep their rows and announce their phase.
/// The audience choice has no row in the reference and uses the same option-row design (OD-P2-11). The creator
/// studio's own bottom bar (live, bazaar, showcase, cart) belongs to later phases and is not shown.
class CreatePostScreen extends ConsumerStatefulWidget {
  const CreatePostScreen({super.key, this.editing});

  /// When set, the screen edits caption and audience of this post (media are fixed after publishing).
  final Post? editing;

  @override
  ConsumerState<CreatePostScreen> createState() => _CreatePostScreenState();
}

class _Slide {
  _Slide({this.bytes, this.media});
  final Uint8List? bytes;
  MediaInfo? media;
  bool failed = false;
}

class _CreatePostScreenState extends ConsumerState<CreatePostScreen> {
  static const maxSlides = 10, captionMax = 2200;
  late final _caption = TextEditingController(text: widget.editing?.caption ?? '');
  late bool _followersOnly = widget.editing?.visibility == 'followers';
  late final List<_Slide> _slides = [for (final m in widget.editing?.media ?? const <MediaInfo>[]) _Slide(media: m)];
  final String _idem = Repo.newIdempotencyKey();
  bool _publishing = false;
  int _uploading = 0;

  bool get _editing => widget.editing != null;
  bool get _ready => (_editing || (_slides.isNotEmpty && _slides.every((s) => s.media != null))) && _uploading == 0 && !_publishing;

  @override
  void dispose() {
    _caption.dispose();
    super.dispose();
  }

  Future<void> _add() async {
    if (_editing) return;
    final room = maxSlides - _slides.length;
    if (room <= 0) {
      toast(context, 'حداکثر ${toPersianDigits(maxSlides)} تصویر برای هر پست.');
      return;
    }
    final picked = await ref.read(imagePickerProvider).pick(max: room);
    for (final p in picked) {
      final s = _Slide(bytes: p.bytes);
      setState(() {
        _slides.add(s);
        _uploading++;
      });
      try {
        s.media = await ref.read(repoProvider).uploadImage(p.bytes, p.name, purpose: 'post');
      } on ApiException catch (e) {
        s.failed = true;
        if (mounted) toast(context, e.title);
      } finally {
        if (mounted) setState(() => _uploading--);
      }
    }
  }

  Future<void> _publish() async {
    if (!_ready) return;
    setState(() => _publishing = true);
    final repo = ref.read(repoProvider);
    try {
      final vis = _followersOnly ? 'followers' : 'public';
      if (_editing) {
        final p = await repo.updatePost(widget.editing!, caption: _caption.text, visibility: vis);
        if (!mounted) return;
        toast(context, 'پست به‌روزرسانی شد.');
        context.pop(p);
      } else {
        final p = await repo.createPost(caption: _caption.text, visibility: vis, mediaIds: [for (final s in _slides) s.media!.id], idempotencyKey: _idem);
        if (!mounted) return;
        toast(context, 'پست منتشر شد.');
        context.go('/home');
        context.push('/post/${p.id}');
      }
    } on ApiException catch (e) {
      if (mounted) toast(context, e.status == 412 ? 'پست در جای دیگری ویرایش شده است؛ دوباره باز کنید.' : e.title);
    } finally {
      if (mounted) setState(() => _publishing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: _StudioHeader(cs: cs),
      body: ListView(padding: const EdgeInsets.fromLTRB(16, 0, 16, 32), children: [
        _card(cs, radius: 32, padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8), child: Row(children: [
          RoundIconButton(icon: Sym.close, label: 'بستن', size: 40, iconSize: 20, background: cs.surfaceContainerLow, onTap: () => context.pop()),
          const SizedBox(width: 4),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(_editing ? 'ویرایش پست' : 'پست جدید', style: tw(22, 28, 600, cs.onSurface)),
              Text('استودیو سازنده ویرا', style: tw(10, 12, 700, cs.onSurfaceVariant)),
            ]),
          ),
          _pill(cs, 'پیش‌نمایش', cs.surfaceContainer, cs.onSurface, null, () => soon(context, 'پیش‌نمایش پست', 'نسخه‌های بعد')),
          const SizedBox(width: 4),
          _pill(cs, _editing ? 'ذخیره' : 'انتشار', cs.primaryContainer, cs.onPrimaryContainer, Sym.send, _ready ? _publish : null, key: const Key('publish-top')),
        ])),
        const SizedBox(height: 16),
        _section(cs, Sym.photoLibrary, cs.primary, 'تصاویر پست',
            trailing: _badge(cs, _slides.isEmpty ? 'بدون تصویر' : '${toPersianDigits(_slides.length)} از ${toPersianDigits(maxSlides)} اسلاید', cs.surfaceContainerHigh, cs.onSurfaceVariant),
            child: _media(cs)),
        const SizedBox(height: 16),
        _section(cs, Sym.editNote, cs.secondary, 'متن کپشن و توضیحات',
            trailing: Text('${toPersianDigits(_caption.text.characters.length)} / ${toPersianDigits(captionMax)}', style: tw(10, 12, 700, cs.onSurfaceVariant)),
            child: Container(
              padding: const EdgeInsets.all(8),
              color: cs.surfaceContainerLow,
              child: Column(children: [
                TextField(
                  key: const Key('caption'),
                  controller: _caption,
                  minLines: 4,
                  maxLines: 8,
                  inputFormatters: [LengthLimitingTextInputFormatter(captionMax)],
                  onChanged: (_) => setState(() {}),
                  style: tw(14, 22.75, 400, cs.onSurface),
                  cursorColor: cs.primaryContainer,
                  decoration: InputDecoration(
                    isCollapsed: true,
                    border: InputBorder.none,
                    hintText: 'دربارهٔ این پست بنویسید… برای برچسب از # استفاده کنید.',
                    hintStyle: tw(14, 22.75, 400, cs.onSurfaceVariant.withValues(alpha: 0.6)),
                  ),
                ),
                Row(children: [
                  RoundIconButton(icon: Sym.sentimentSatisfied, label: 'ایموجی', size: 32, iconSize: 18, color: cs.onSurfaceVariant, onTap: () => soon(context, 'صفحه‌کلید ایموجی', 'نسخه‌های بعد')),
                  RoundIconButton(icon: Sym.alternateEmail, label: 'منشن', size: 32, iconSize: 18, color: cs.onSurfaceVariant, onTap: () => soon(context, 'منشن', 'فاز ۵')),
                ]),
              ]),
            )),
        const SizedBox(height: 16),
        _section(cs, Sym.shoppingCartCheckout, cs.primaryContainer, 'کالاهای متصل به خرید مستقیم',
            subtitle: 'تگ هوشمند در سبد خرید آنی',
            trailing: _badge(cs, 'فاز ۴', cs.primaryFixed, cs.primary),
            child: Material(
              color: cs.surfaceContainer,
              borderRadius: BorderRadius.circular(999),
              child: InkWell(
                borderRadius: BorderRadius.circular(999),
                onTap: () => soon(context, 'تگ کالا در پست', 'فاز ۴'),
                child: SizedBox(
                  height: 48,
                  child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                    Sy(Sym.addCircle, size: 20, color: cs.secondary),
                    const SizedBox(width: 4),
                    Text('+ تگ کردن کالا از انبار فروشگاه', style: tw(14, 18, 600, cs.secondary)),
                  ]),
                ),
              ),
            )),
        const SizedBox(height: 16),
        _section(cs, Sym.tune, cs.secondaryContainer, 'تنظیمات انتشار و رفتار پست',
            child: Column(children: [
              _option(cs, Sym.group, cs.tertiaryContainer, cs.onTertiaryContainer, 'فقط برای دنبال‌کنندگان', 'پست در فید عمومی و کاوش نمایش داده نمی‌شود',
                  ArchiveSwitch(key: const Key('followers-only'), value: _followersOnly, label: 'فقط برای دنبال‌کنندگان', onChanged: (v) => setState(() => _followersOnly = v))),
              _option(cs, Sym.autoStories, cs.secondaryFixed, cs.onSecondaryFixed, 'اشتراک‌گذاری در استوری خودکار', 'بازنشر فوری در استوری (فاز ۳)', _off('استوری خودکار', 'فاز ۳')),
              _option(cs, Sym.touchApp, cs.secondaryContainer, cs.onSecondaryContainer, 'خرید مستقیم با ۱ کلیک', 'فعال با راه‌اندازی فروشگاه (فاز ۴)', _off('خرید مستقیم', 'فاز ۴')),
              _option(cs, Sym.commentsDisabled, cs.surfaceContainerHigh, cs.onSurfaceVariant, 'بستن دیدگاه‌ها', 'از تنظیمات حریم خصوصی برای همه پست‌ها', _off('بستن دیدگاه‌های یک پست', 'نسخه‌های بعد')),
              _option(cs, Sym.pinDrop, cs.tertiaryContainer, cs.onTertiaryContainer, 'مکان', 'افزودن مکان در نسخه‌های بعد', RoundIconButton(icon: Sym.navigateBefore, label: 'مکان', size: 32, iconSize: 18, background: cs.surfaceContainerHigh, onTap: () => soon(context, 'مکان پست', 'نسخه‌های بعد'))),
            ])),
        const SizedBox(height: 16),
        Opacity(
          opacity: _ready ? 1 : 0.5,
          child: Material(
            color: cs.primaryContainer,
            borderRadius: BorderRadius.circular(999),
            child: InkWell(
              key: const Key('publish'),
              borderRadius: BorderRadius.circular(999),
              onTap: _ready ? _publish : null,
              child: SizedBox(
                height: 56,
                child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                  Sy(Sym.rocketLaunch, size: 24, color: cs.onPrimaryContainer),
                  const SizedBox(width: 4),
                  Text(_publishing ? 'در حال انتشار…' : (_editing ? 'ذخیره تغییرات پست' : 'انتشار در فید'), style: tw(18, 24, 700, cs.onPrimaryContainer)),
                ]),
              ),
            ),
          ),
        ),
        const SizedBox(height: 4),
        Material(
          color: cs.surfaceContainer,
          borderRadius: BorderRadius.circular(999),
          child: InkWell(
            borderRadius: BorderRadius.circular(999),
            onTap: () => soon(context, 'پیش‌نویس‌ها', 'نسخه‌های بعد'),
            child: SizedBox(
              height: 44,
              child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                Sy(Sym.bookmarkBorder, size: 18, color: cs.onSurfaceVariant),
                const SizedBox(width: 4),
                Text('ذخیره در پیش‌نویس‌ها', style: tw(14, 18, 600, cs.onSurfaceVariant)),
              ]),
            ),
          ),
        ),
      ]),
    );
  }

  Widget _off(String what, String phase) =>
      GestureDetector(onTap: () => soon(context, what, phase), child: AbsorbPointer(child: ArchiveSwitch(value: false, onChanged: null, label: what)));

  Widget _media(ColorScheme cs) {
    Widget tile(int i, {double? h}) {
      final s = _slides[i];
      return SizedBox(
        height: h,
        child: Stack(fit: StackFit.expand, children: [
          ColoredBox(
            color: cs.surfaceContainer,
            child: s.media != null
                ? NetImage(s.media!.url)
                : (s.bytes != null ? Image.memory(s.bytes!, fit: BoxFit.cover) : null),
          ),
          if (s.media == null && !s.failed) const Center(child: CircularProgressIndicator(strokeWidth: 2)),
          if (s.failed) Center(child: Sy(Sym.error, size: 28, color: cs.error)),
          if (i == 0)
            const DecoratedBox(decoration: BoxDecoration(gradient: LinearGradient(begin: Alignment.bottomCenter, end: Alignment.topCenter, colors: [Color(0x99171B26), Color(0x00000000), Color(0x00000000)]))),
          if (i == 0)
            Positioned(
              right: 4,
              bottom: 4,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(color: cs.onSurface.withValues(alpha: 0.4), borderRadius: BorderRadius.circular(999)),
                child: Text('اصلی (کاور)', style: tw(10, 12, 700, cs.surface)),
              ),
            )
          else
            Positioned(
              right: 4,
              top: 4,
              child: Container(
                width: 20,
                height: 20,
                decoration: BoxDecoration(color: Colors.white.withValues(alpha: 0.9), shape: BoxShape.circle),
                alignment: Alignment.center,
                child: Text(toPersianDigits(i + 1), style: tw(10, 12, 700, const Color(0xFF171B26))),
              ),
            ),
          if (!_editing)
            Positioned(
              left: 4,
              top: 4,
              child: RoundIconButton(
                icon: Sym.close,
                label: 'حذف تصویر',
                size: 28,
                iconSize: 16,
                background: Colors.white.withValues(alpha: 0.8),
                color: const Color(0xFF171B26),
                onTap: () => setState(() => _slides.removeAt(i)),
              ),
            ),
        ]),
      );
    }

    final add = GestureDetector(
      key: const Key('add-slide'),
      onTap: _add,
      child: Container(
        color: cs.surfaceContainerLow,
        child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Sy(Sym.addPhotoAlternate, size: 24, color: cs.primary),
          const SizedBox(height: 4),
          Text('+ افزودن اسلاید', style: tw(10, 12, 700, cs.onSurfaceVariant)),
        ]),
      ),
    );
    if (_slides.isEmpty) return SizedBox(height: 270, child: add);
    return SizedBox(
      height: 270,
      child: Row(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Expanded(flex: 2, child: tile(0)),
        const SizedBox(width: 4),
        Expanded(
          child: ListView(children: [
            for (var i = 1; i < _slides.length; i++) ...[tile(i, h: 106), const SizedBox(height: 4)],
            if (!_editing && _slides.length < maxSlides) SizedBox(height: 106, child: add),
          ]),
        ),
      ]),
    );
  }

  Widget _card(ColorScheme cs, {required Widget child, double radius = 32, EdgeInsets padding = const EdgeInsets.all(16)}) => Container(
        padding: padding,
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(radius),
          boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
        ),
        child: child,
      );

  Widget _section(ColorScheme cs, IconData icon, Color iconColor, String title, {String? subtitle, Widget? trailing, required Widget child}) => _card(
        cs,
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Row(children: [
            Sy(icon, size: 20, color: iconColor),
            const SizedBox(width: 4),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(title, style: tw(16, 22, 600, cs.onSurface)),
                if (subtitle != null) Text(subtitle, style: tw(10, 12, 700, cs.onSurfaceVariant)),
              ]),
            ),
            ?trailing,
          ]),
          const SizedBox(height: 8),
          child,
        ]),
      );

  Widget _badge(ColorScheme cs, String text, Color bg, Color fg) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
        decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(999)),
        child: Text(text, style: tw(10, 12, 500, fg)),
      );

  Widget _pill(ColorScheme cs, String label, Color bg, Color fg, IconData? icon, VoidCallback? onTap, {Key? key}) => Opacity(
        opacity: onTap == null ? 0.5 : 1,
        child: Material(
          color: bg,
          borderRadius: BorderRadius.circular(999),
          child: InkWell(
            key: key,
            borderRadius: BorderRadius.circular(999),
            onTap: onTap,
            child: Container(
              height: 36,
              padding: const EdgeInsets.symmetric(horizontal: 16),
              alignment: Alignment.center,
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                Text(label, style: tw(12, 16, 600, fg)),
                if (icon != null) ...[const SizedBox(width: 4), Sy(icon, size: 16, color: fg)],
              ]),
            ),
          ),
        ),
      );

  Widget _option(ColorScheme cs, IconData icon, Color discBg, Color discFg, String title, String sub, Widget trailing) => Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.all(4),
        color: cs.surfaceContainerLow.withValues(alpha: 0.7),
        child: Row(children: [
          Container(width: 36, height: 36, decoration: BoxDecoration(color: discBg, shape: BoxShape.circle), alignment: Alignment.center, child: Sy(icon, size: 18, color: discFg)),
          const SizedBox(width: 8),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(title, style: tw(16, 22, 600, cs.onSurface)),
              Text(sub, style: tw(10, 12, 700, cs.onSurfaceVariant)),
            ]),
          ),
          const SizedBox(width: 8),
          trailing,
        ]),
      );
}

class _StudioHeader extends StatelessWidget implements PreferredSizeWidget {
  const _StudioHeader({required this.cs});
  final ColorScheme cs;

  @override
  Size get preferredSize => const Size.fromHeight(64);

  @override
  Widget build(BuildContext context) => Container(
        decoration: BoxDecoration(color: cs.surface.withValues(alpha: 0.8), boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 1))]),
        child: SafeArea(
          bottom: false,
          child: SizedBox(
            height: 64,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(children: [
                Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(color: cs.primary, shape: BoxShape.circle),
                  alignment: Alignment.center,
                  child: Text('V', style: tw(22, 28, 600, cs.onPrimary)),
                ),
                const SizedBox(width: 8),
                Expanded(child: Text('استودیو سازنده', style: tw(22, 28, 600, cs.onSurface))),
                RoundIconButton(icon: Sym.notifications, label: 'فعالیت‌ها', color: cs.onSurfaceVariant, onTap: () => context.push('/activity')),
                HeaderPersonChip(onTap: () => context.go('/profile')),
              ]),
            ),
          ),
        ),
      );
}
