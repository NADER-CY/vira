import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/format/persian.dart';
import '../../core/media/picker.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/states.dart';

/// Edit profile. Reference: `_17` (M-017, dark; the light business editor `_146` is a different, seller-oriented
/// layout — conflict OD-P2-05, `_17` used for both themes). Phase 2 fields: picture, name, handle, bio, link.
/// The reference's pronoun, category, e-mail, gender and business toggles keep their place and say when they
/// arrive; the linked phone is the real (masked) number.
class EditProfileScreen extends ConsumerStatefulWidget {
  const EditProfileScreen({super.key});

  @override
  ConsumerState<EditProfileScreen> createState() => _EditProfileScreenState();
}

class _EditProfileScreenState extends ConsumerState<EditProfileScreen> {
  static const bioMax = 300;
  Profile? _p;
  Account? _account;
  Object? _error;
  final _name = TextEditingController();
  final _handle = TextEditingController();
  final _bio = TextEditingController();
  final _website = TextEditingController();
  bool _editingLink = false;
  String? _avatarId;
  String? _avatarUrl;
  bool _uploading = false, _saving = false;
  Map<String, String> _fieldErrors = const {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in [_name, _handle, _bio, _website]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final repo = ref.read(repoProvider);
      final p = await repo.myProfile();
      final a = await repo.account();
      if (!mounted) return;
      setState(() {
        _p = p;
        _account = a;
        _name.text = p.author.displayName;
        _handle.text = p.author.handle;
        _bio.text = p.bio;
        _website.text = p.website;
        _avatarUrl = p.author.avatarUrl;
        _error = null;
      });
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  Future<void> _pickAvatar() async {
    final picked = await ref.read(imagePickerProvider).pick();
    if (picked.isEmpty) return;
    setState(() => _uploading = true);
    try {
      final m = await ref.read(repoProvider).uploadImage(picked.first.bytes, picked.first.name, purpose: 'avatar');
      setState(() {
        _avatarId = m.id;
        _avatarUrl = m.url;
      });
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _save() async {
    final p = _p;
    if (p == null || _saving) return;
    final changes = <String, Object?>{
      if (_name.text.trim() != p.author.displayName) 'display_name': _name.text.trim(),
      if (_handle.text.trim() != p.author.handle) 'handle': _handle.text.trim(),
      if (_bio.text != p.bio) 'bio': _bio.text,
      if (_website.text.trim() != p.website) 'website': _website.text.trim(),
      'avatar_media_id': ?_avatarId,
    };
    if (changes.isEmpty) {
      context.pop();
      return;
    }
    setState(() {
      _saving = true;
      _fieldErrors = const {};
    });
    try {
      final updated = await ref.read(repoProvider).updateProfile(p.version, changes);
      ref.read(sessionProvider).updateHandle(updated.author.handle);
      if (!mounted) return;
      toast(context, 'نمایه ذخیره شد.');
      context.pop(updated);
    } on ApiException catch (e) {
      setState(() => _fieldErrors = e.fieldErrors);
      if (!mounted) return;
      if (e.status == 412 || e.code == 'conflict' || e.code == 'precondition_failed') {
        toast(context, 'نمایه در جای دیگری تغییر کرده است؛ دوباره بارگذاری شد.');
        await _load();
      } else {
        toast(context, e.fieldErrors.isEmpty ? e.title : 'برخی فیلدها نیاز به اصلاح دارند.');
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  String? _err(String field) => switch (_fieldErrors[field]) {
        null => null,
        'taken' => 'این نام کاربری قبلاً انتخاب شده است.',
        'invalid' || 'format' => field == 'handle' ? 'فقط حروف انگلیسی، عدد، نقطه و زیرخط (۳ تا ۳۰ نویسه).' : 'مقدار معتبر نیست.',
        'too_long' => 'طولانی‌تر از حد مجاز است.',
        'required' => 'این فیلد الزامی است.',
        'reserved' => 'این نام کاربری رزرو شده است.',
        final c => 'مقدار معتبر نیست ($c).',
      };

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return Scaffold(
      backgroundColor: cs.surface,
      appBar: ArchiveHeader(
        title: 'ویرایش نمایه',
        horizontal: 16,
        shadow: const BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 1)),
        actions: [
          RoundIconButton(icon: Sym.moreVert, label: 'گزینه‌های بیشتر', iconSize: 20, onTap: () => context.push('/settings')),
          const HeaderPersonChip(),
        ],
      ),
      body: _p == null
          ? (_error == null ? const LoadingView(rows: 5) : Center(child: ErrorView(error: _error!, onRetry: _load)))
          : ListView(padding: const EdgeInsets.fromLTRB(12, 0, 12, 32), children: [
              _avatar(cs),
              const SizedBox(height: 16),
              _section(cs, icon: Sym.badge, iconColor: cs.primary, title: 'اطلاعات هویتی نمایه', trailing: 'نمایش عمومی', children: [
                _field(cs, label: 'نام و نام خانوادگی', hint: 'الزامی', child: _input(cs, _name, icon: Sym.person, key: 'name', maxLength: 50), error: _err('display_name')),
                _field(cs, label: 'نام کاربری (آیدی اختصاصی)', hint: 'انگلیسی', error: _err('handle'), child: _input(
                  cs,
                  _handle,
                  icon: Sym.alternateEmail,
                  ltr: true,
                  key: 'handle',
                  big: true,
                  maxLength: 30,
                  formatters: [FilteringTextInputFormatter.allow(RegExp(r'[A-Za-z0-9._]'))],
                )),
                _field(cs, label: 'ضمیر یا عنوان مورد خطاب', hint: 'اختیاری', child: GestureDetector(
                  onTap: () => soon(context, 'ضمیر و عنوان خطاب', 'نسخه‌های بعد'),
                  child: AbsorbPointer(child: _input(cs, TextEditingController(), icon: Sym.recordVoiceOver, placeholder: 'مثال: او / ایشان، خانم طراح')),
                )),
                _bioField(cs),
                _links(cs),
              ]),
              const SizedBox(height: 12),
              _section(cs, icon: Sym.storefront, iconColor: cs.secondary, title: 'دسته‌بندی و اطلاعات تماس', trailing: 'امنیت داده‌ها', children: [
                _field(cs, label: 'دسته‌بندی موضوعی نمایه', child: _rowButton(cs, Sym.checkroom, 'انتخاب دسته‌بندی', () => soon(context, 'دسته‌بندی نمایه', 'فاز ۴'))),
                _phone(cs),
                _field(cs, label: 'پست الکترونیکی کاری', child: GestureDetector(
                  onTap: () => soon(context, 'ایمیل کاری', 'فاز ۴'),
                  child: AbsorbPointer(child: _input(cs, TextEditingController(), icon: Sym.mail, ltr: true, placeholder: 'name@example.com')),
                )),
                _field(cs, label: 'جنسیت', child: Row(children: [
                  Expanded(child: _choice(cs, Sym.female, 'خانم')),
                  const SizedBox(width: 8),
                  Expanded(child: _choice(cs, Sym.male, 'آقا / ترجیح عدم پاسخ')),
                ])),
              ]),
              const SizedBox(height: 12),
              _section(cs, icon: Sym.toggleOn, iconColor: cs.primary, title: 'تنظیمات حرفه‌ای نمایه ویرا', trailing: 'ویژه کسب‌وکار', trailingColor: cs.primary, children: [
                _toggleRow(cs, 'نمایش برچسب دسته‌بندی در پروفایل', 'نمایش عنوان تجاری شما در زیر نام نمایشی'),
                _toggleRow(cs, 'نمایش دکمه تماس و آدرس در نمایه', 'فعال‌سازی تماس سریع مستقیم برای خریداران'),
                _businessCard(cs),
              ]),
              const SizedBox(height: 20),
              Material(
                color: cs.primaryContainer,
                borderRadius: BorderRadius.circular(8),
                elevation: 0,
                child: InkWell(
                  key: const Key('save-profile'),
                  borderRadius: BorderRadius.circular(8),
                  onTap: _saving || _uploading ? null : _save,
                  child: SizedBox(
                    height: 48,
                    child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
                      const Sy(Sym.checkCircle, size: 20, color: Colors.white),
                      const SizedBox(width: 4),
                      Text(_saving ? 'در حال ذخیره…' : 'ذخیره تغییرات نمایه', style: tw(14, 20, 700, Colors.white)),
                    ]),
                  ),
                ),
              ),
              const SizedBox(height: 4),
              Text('تغییرات به صورت آنی در نمایه عمومی شما اعمال خواهد شد', style: tw(11, 14, 500, cs.tertiary), textAlign: TextAlign.center),
            ]),
    );
  }

  Widget _avatar(ColorScheme cs) => SizedBox(
        height: 156,
        child: Stack(clipBehavior: Clip.none, alignment: Alignment.topCenter, children: [
          Positioned(
            top: -8,
            child: Container(
              width: 128,
              height: 128,
              decoration: BoxDecoration(shape: BoxShape.circle, color: cs.primaryContainer.withValues(alpha: 0.2), boxShadow: [BoxShadow(color: cs.primaryContainer.withValues(alpha: 0.2), blurRadius: 40)]),
            ),
          ),
          Positioned(
            top: 12,
            child: SizedBox(
              width: 96,
              height: 96,
              child: Stack(children: [
                Container(
                  width: 96,
                  height: 96,
                  padding: const EdgeInsets.all(2),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: LinearGradient(begin: Alignment.bottomLeft, end: Alignment.topRight, colors: [cs.primaryContainer, cs.surfaceContainerHigh, cs.secondaryContainer]),
                    boxShadow: const [BoxShadow(color: Color(0x1A000000), blurRadius: 25, offset: Offset(0, 20), spreadRadius: -5)],
                  ),
                  child: ClipOval(
                    child: ColoredBox(
                      color: cs.surfaceContainerLow,
                      child: _uploading
                          ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
                          : (_avatarUrl == null || _avatarUrl!.isEmpty ? Center(child: Sy(Sym.person, size: 40, color: cs.onSurfaceVariant)) : NetImage(_avatarUrl)),
                    ),
                  ),
                ),
                Positioned(
                  left: 0,
                  bottom: 0,
                  child: RoundIconButton(
                    key: const Key('change-avatar'),
                    icon: Sym.photoCamera,
                    label: 'تغییر عکس نمایه',
                    size: 32,
                    iconSize: 18,
                    color: cs.onPrimaryFixed,
                    background: cs.primaryContainer,
                    onTap: _pickAvatar,
                  ),
                ),
              ]),
            ),
          ),
          Positioned(
            top: 116,
            child: Column(children: [
              GestureDetector(onTap: _pickAvatar, child: Text('تغییر تصویر نمایه یا نشان تجاری', style: tw(12, 16, 600, cs.primary))),
              const SizedBox(height: 6),
              Text('اندازه پیشنهادی: ۵۰۰ در ۵۰۰ پیکسل', style: tw(11, 14, 500, cs.tertiary)),
            ]),
          ),
        ]),
      );

  Widget _section(ColorScheme cs, {required IconData icon, required Color iconColor, required String title, required String trailing, Color? trailingColor, required List<Widget> children}) =>
      Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: cs.surfaceContainerLow,
          borderRadius: BorderRadius.circular(12),
          boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          Row(children: [
            Sy(icon, size: 18, color: iconColor),
            const SizedBox(width: 4),
            Expanded(child: Text(title, style: tw(14, 20, 600, cs.onSurface))),
            Text(trailing, style: tw(11, 14, trailingColor == null ? 500 : 600, trailingColor ?? cs.tertiary)),
          ]),
          for (final c in children) ...[const SizedBox(height: 12), c],
        ]),
      );

  Widget _field(ColorScheme cs, {required String label, String? hint, required Widget child, String? error}) => Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Row(children: [
          Expanded(child: Text(label, style: tw(12, 16, 600, cs.onSurfaceVariant))),
          if (hint != null) Text(hint, style: tw(11, 14, 500, cs.tertiary)),
        ]),
        const SizedBox(height: 4),
        child,
        if (error != null) Padding(padding: const EdgeInsets.only(top: 4), child: Text(error, style: tw(11, 14, 500, cs.error))),
      ]);

  Widget _input(ColorScheme cs, TextEditingController c,
      {required IconData icon, bool ltr = false, String? key, String? placeholder, bool big = false, int? maxLength, List<TextInputFormatter>? formatters}) {
    final style = big ? tw(18, 24, 600, cs.onSurface, spacing: 0.4) : tw(14, 20, 400, cs.onSurface);
    return Container(
      height: 48,
      decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(8)),
      child: Directionality(
        textDirection: ltr ? TextDirection.ltr : TextDirection.rtl,
        child: Row(children: [
          const SizedBox(width: 12),
          Sy(icon, size: ltr ? 18 : 20, color: cs.tertiary),
          const SizedBox(width: 8),
          Expanded(
            child: TextField(
              key: key == null ? null : Key('field-$key'),
              controller: c,
              style: style,
              cursorColor: cs.primaryContainer,
              inputFormatters: [...?formatters, if (maxLength != null) LengthLimitingTextInputFormatter(maxLength)],
              onChanged: (_) => setState(() {}),
              decoration: InputDecoration(
                isCollapsed: true,
                border: InputBorder.none,
                hintText: placeholder,
                hintStyle: tw(14, 20, 400, cs.tertiaryContainer),
              ),
            ),
          ),
          const SizedBox(width: 12),
        ]),
      ),
    );
  }

  Widget _bioField(ColorScheme cs) => Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Row(children: [
          Expanded(child: Text('معرفی و بیوگرافی (Bio)', style: tw(12, 16, 600, cs.onSurfaceVariant))),
          Text('${toPersianDigits(_bio.text.characters.length)} / ${toPersianDigits(bioMax)}', style: tw(11, 14, 500, cs.primary)),
        ]),
        const SizedBox(height: 4),
        Container(
          decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(8)),
          padding: const EdgeInsets.all(12),
          child: TextField(
            key: const Key('field-bio'),
            controller: _bio,
            minLines: 3,
            maxLines: 5,
            inputFormatters: [LengthLimitingTextInputFormatter(bioMax)],
            onChanged: (_) => setState(() {}),
            style: tw(14, 22.75, 400, cs.onSurface),
            cursorColor: cs.primaryContainer,
            decoration: const InputDecoration(isCollapsed: true, border: InputBorder.none),
          ),
        ),
        if (_err('bio') != null) Padding(padding: const EdgeInsets.only(top: 4), child: Text(_err('bio')!, style: tw(11, 14, 500, cs.error))),
      ]);

  Widget _links(ColorScheme cs) {
    final hasLink = _website.text.trim().isNotEmpty && !_editingLink;
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      Padding(padding: const EdgeInsets.only(top: 4), child: Text('پیوندها و وب‌سایت رسمی', style: tw(12, 16, 600, cs.onSurfaceVariant))),
      const SizedBox(height: 8),
      if (hasLink)
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(8)),
          child: Row(children: [
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(color: cs.surfaceContainerHigh, shape: BoxShape.circle),
              alignment: Alignment.center,
              child: Sy(Sym.link, size: 18, color: cs.primary),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text(isolate(_website.text.trim()), style: tw(12, 16, 600, cs.onSurface), overflow: TextOverflow.ellipsis),
                Text('پیوند نمایه', style: tw(11, 14, 500, cs.tertiary)),
              ]),
            ),
            RoundIconButton(
              key: const Key('remove-link'),
              icon: Sym.close,
              label: 'حذف پیوند',
              size: 32,
              iconSize: 20,
              color: cs.tertiary,
              onTap: () => setState(() => _website.clear()),
            ),
          ]),
        )
      else if (_editingLink)
        _input(cs, _website, icon: Sym.link, ltr: true, key: 'website', placeholder: 'https://example.com', maxLength: 200),
      if (_err('website') != null) Padding(padding: const EdgeInsets.only(top: 4), child: Text(_err('website')!, style: tw(11, 14, 500, cs.error))),
      if (!hasLink && !_editingLink || _editingLink) const SizedBox(height: 8),
      Material(
        color: cs.surfaceContainer,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          key: const Key('add-link'),
          borderRadius: BorderRadius.circular(8),
          onTap: () => setState(() => _editingLink = !_editingLink),
          child: SizedBox(
            height: 34,
            child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              Sy(_editingLink ? Sym.check : Sym.addCircle, size: 18, color: cs.primary),
              const SizedBox(width: 4),
              Text(_editingLink ? 'تأیید پیوند' : (hasLink ? 'تغییر پیوند' : 'افزودن لینک یا کانال تلگرام'), style: tw(12, 16, 600, cs.primary)),
            ]),
          ),
        ),
      ),
    ]);
  }

  Widget _rowButton(ColorScheme cs, IconData icon, String label, VoidCallback onTap) => Material(
        color: cs.surfaceContainer,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: onTap,
          child: SizedBox(
            height: 48,
            child: Row(children: [
              const SizedBox(width: 12),
              Sy(icon, size: 20, color: cs.primary),
              const SizedBox(width: 8),
              Expanded(child: Text(label, style: tw(14, 20, 600, cs.onSurface))),
              Sy(Sym.arrowBackIos, size: 20, color: cs.tertiary),
              const SizedBox(width: 12),
            ]),
          ),
        ),
      );

  Widget _phone(ColorScheme cs) {
    final masked = _account?.phoneMasked ?? '';
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      Row(children: [
        Expanded(child: Text('تلفن همراه متصل', style: tw(12, 16, 600, cs.onSurfaceVariant))),
        Text('تایید شده', style: tw(11, 14, 500, cs.primary)),
        const SizedBox(width: 4),
        Sy(Sym.verified, size: 14, color: cs.primary),
      ]),
      const SizedBox(height: 4),
      Container(
        height: 48,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(8)),
        child: Directionality(
          textDirection: TextDirection.ltr,
          child: Row(children: [
            Sy(Sym.call, size: 18, color: cs.tertiary),
            const SizedBox(width: 4),
            Expanded(child: Text(toPersianDigits(masked), key: const Key('masked-phone'), style: tw(18, 24, 600, cs.onSurface, spacing: 0.9))),
            GestureDetector(onTap: () => soon(context, 'تغییر شماره', 'فاز ۴'), child: Text('تغییر شماره', style: tw(11, 14, 500, cs.secondary))),
          ]),
        ),
      ),
    ]);
  }

  Widget _choice(ColorScheme cs, IconData icon, String label) => GestureDetector(
        onTap: () => soon(context, 'جنسیت', 'نسخه‌های بعد'),
        child: Container(
          height: 44,
          decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(8)),
          child: Row(mainAxisAlignment: MainAxisAlignment.center, children: [
            Sy(icon, size: 18, color: cs.tertiary),
            const SizedBox(width: 4),
            Flexible(child: Text(label, style: tw(12, 16, 600, cs.tertiary), overflow: TextOverflow.ellipsis)),
          ]),
        ),
      );

  Widget _toggleRow(ColorScheme cs, String title, String sub) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(children: [
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(title, style: tw(12, 16, 600, cs.onSurface)),
              const SizedBox(height: 2),
              Text(sub, style: tw(11, 14, 500, cs.tertiary)),
            ]),
          ),
          GestureDetector(onTap: () => soon(context, 'تنظیمات حرفه‌ای', 'فاز ۴'), child: AbsorbPointer(child: ArchiveSwitch(value: false, onChanged: null, label: title))),
        ]),
      );

  Widget _businessCard(ColorScheme cs) => GestureDetector(
        onTap: () => soon(context, 'حساب تجاری', 'فاز ۴'),
        child: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            gradient: LinearGradient(colors: [cs.surfaceContainer, cs.surfaceContainerHigh]),
            boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
          ),
          child: Row(children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(color: cs.secondaryContainer.withValues(alpha: 0.4), shape: BoxShape.circle),
              alignment: Alignment.center,
              child: Sy(Sym.verified, size: 24, color: cs.secondary, fill: true),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('حساب تجاری و حرفه‌ای ویرا', style: tw(12, 16, 600, cs.onSurface)),
                Text('ارتقای حساب به فروشگاه رسمی در فاز ۴', style: tw(11, 14, 500, cs.secondary), overflow: TextOverflow.ellipsis),
              ]),
            ),
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(color: cs.surfaceContainerHighest, shape: BoxShape.circle),
              alignment: Alignment.center,
              child: Sy(Sym.settings, size: 18, color: cs.onSurface),
            ),
          ]),
        ),
      );
}
