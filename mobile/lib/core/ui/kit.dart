import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../api/models.dart';
import '../providers.dart';
import '../theme/symbols.g.dart';
import '../theme/tokens.g.dart';
import '../theme/vira_theme.dart';

/// Building blocks shared by the screens ported from the UI archive. Sizes are the archive's computed CSS values
/// (measured from each reference at 390 px width), so every widget takes explicit pixel values.

/// A Tailwind text style: size / line-height in px, CSS weight, colour.
TextStyle tw(double size, double line, int weight, Color color, {double? spacing}) => TextStyle(
      fontFamily: ViraTokens.fontFamily,
      fontSize: size,
      height: line / size,
      fontWeight: FontWeight.values[(weight ~/ 100) - 1],
      color: color,
      letterSpacing: spacing,
      leadingDistribution: TextLeadingDistribution.even,
    );

/// Material Symbols glyph (`material-symbols-outlined`), optionally filled (`'FILL' 1`).
class Sy extends StatelessWidget {
  const Sy(this.icon, {super.key, this.size = 24, this.color, this.fill = false, this.semantic});
  final IconData icon;
  final double size;
  final Color? color;
  final bool fill;
  final String? semantic;

  @override
  Widget build(BuildContext context) =>
      Icon(icon, size: size, color: color, fill: fill ? 1 : 0, weight: 400, grade: 0, opticalSize: 24, semanticLabel: semantic);
}

/// Shows that an entry kept from the archive is not active yet (doc 09 §5 "feature disabled").
void soon(BuildContext context, String what, String phase) {
  ScaffoldMessenger.of(context)
    ..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(content: Text('$what در $phase فعال می‌شود.')));
}

void toast(BuildContext context, String message) {
  ScaffoldMessenger.of(context)
    ..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(content: Text(message)));
}

/// Round icon button as drawn in archive headers (`w-11 h-11 rounded-full`).
class RoundIconButton extends StatelessWidget {
  const RoundIconButton({super.key, required this.icon, required this.onTap, required this.label, this.size = 44, this.iconSize = 24, this.color, this.background, this.border, this.fill = false, this.badge});
  final IconData icon;
  final VoidCallback? onTap;
  final String label;
  final double size, iconSize;
  final Color? color, background;
  final BoxBorder? border;
  final bool fill;
  final Widget? badge;

  @override
  Widget build(BuildContext context) => Semantics(
        button: true,
        label: label,
        child: SizedBox(
          width: size,
          height: size,
          child: Stack(clipBehavior: Clip.none, children: [
            Positioned.fill(
              child: Material(
                color: background ?? Colors.transparent,
                shape: const CircleBorder(),
                child: InkWell(
                  customBorder: const CircleBorder(),
                  onTap: onTap,
                  child: Container(
                    decoration: border == null ? null : BoxDecoration(shape: BoxShape.circle, border: border),
                    alignment: Alignment.center,
                    child: ExcludeSemantics(child: Sy(icon, size: iconSize, color: color ?? context.cs.onSurface, fill: fill)),
                  ),
                ),
              ),
            ),
            ?badge,
          ]),
        ),
      );
}

/// The person chip on the end of archive headers (`w-8 h-8 rounded-full bg-primary` + `person`).
class HeaderPersonChip extends StatelessWidget {
  const HeaderPersonChip({super.key, this.onTap});
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) => Semantics(
        button: true,
        label: 'حساب من',
        child: GestureDetector(
          onTap: onTap,
          child: Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(color: context.cs.primary, shape: BoxShape.circle),
            alignment: Alignment.center,
            child: Sy(Sym.person, size: 18, color: context.cs.onPrimary),
          ),
        ),
      );
}

/// Archive top bar: back (`arrow_forward` — RTL), title, trailing actions; translucent surface with a soft shadow.
class ArchiveHeader extends StatelessWidget implements PreferredSizeWidget {
  const ArchiveHeader({
    super.key,
    required this.title,
    this.actions = const [],
    this.height = 56,
    this.horizontal = 16,
    this.centerTitle = false,
    this.titleStyle,
    this.showBack = true,
    this.backgroundAlpha = 0.85,
    this.shadow = const BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 1)),
    this.backSize = 44,
    this.onBack,
    this.backIcon = Sym.arrowForward,
  });

  final IconData backIcon;

  final String title;
  final List<Widget> actions;
  final double height, horizontal, backgroundAlpha, backSize;
  final bool centerTitle, showBack;
  final TextStyle? titleStyle;
  final BoxShadow shadow;
  final VoidCallback? onBack;

  @override
  Size get preferredSize => Size.fromHeight(height);

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final canPop = Navigator.of(context).canPop();
    final style = titleStyle ?? tw(18, 24, 600, cs.onSurface);
    final back = showBack && (canPop || onBack != null)
        ? RoundIconButton(icon: backIcon, label: 'بازگشت', size: backSize, onTap: onBack ?? () => Navigator.of(context).maybePop())
        : null;
    return Container(
      decoration: BoxDecoration(color: cs.surface.withValues(alpha: backgroundAlpha), boxShadow: [shadow]),
      child: SafeArea(
        bottom: false,
        child: SizedBox(
          height: height,
          child: Padding(
            padding: EdgeInsets.symmetric(horizontal: horizontal),
            child: centerTitle
                ? Stack(alignment: Alignment.center, children: [
                    Align(alignment: AlignmentDirectional.centerStart, child: back ?? const SizedBox()),
                    ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 200),
                      child: Text(title, style: style, maxLines: 1, overflow: TextOverflow.ellipsis, textAlign: TextAlign.center),
                    ),
                    Align(alignment: AlignmentDirectional.centerEnd, child: Row(mainAxisSize: MainAxisSize.min, children: actions)),
                  ])
                : Row(children: [
                    if (back != null) ...[back, const SizedBox(width: 8)],
                    Expanded(child: Text(title, style: style, maxLines: 1, overflow: TextOverflow.ellipsis)),
                    ...actions,
                  ]),
          ),
        ),
      ),
    );
  }
}

/// The archive's pill switch (`w-12 h-6 rounded-full p-0.5`, knob `w-5 h-5 bg-white`). In RTL the "on" knob sits
/// at the start (right), exactly as the archive draws `translate-x-0`.
class ArchiveSwitch extends StatelessWidget {
  const ArchiveSwitch({super.key, required this.value, required this.onChanged, required this.label, this.onColor, this.offColor, this.width = 48, this.height = 24});
  final bool value;
  final ValueChanged<bool>? onChanged;
  final String label;
  final Color? onColor, offColor;
  final double width, height;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final knob = height - 4;
    return Semantics(
      toggled: value,
      label: label,
      enabled: onChanged != null,
      child: GestureDetector(
        onTap: onChanged == null ? null : () => onChanged!(!value),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          width: width,
          height: height,
          padding: const EdgeInsets.all(2),
          decoration: BoxDecoration(
            color: value ? (onColor ?? cs.primaryContainer) : (offColor ?? cs.surfaceContainerHighest),
            borderRadius: BorderRadius.circular(height),
          ),
          child: AnimatedAlign(
            duration: const Duration(milliseconds: 150),
            alignment: value ? AlignmentDirectional.centerStart : AlignmentDirectional.centerEnd,
            child: Container(
              width: knob,
              height: knob,
              decoration: const BoxDecoration(color: Colors.white, shape: BoxShape.circle, boxShadow: [BoxShadow(color: Color(0x1A000000), blurRadius: 3, offset: Offset(0, 1))]),
            ),
          ),
        ),
      ),
    );
  }
}

/// Square check box as the archive draws native `<input type=checkbox class="w-5 h-5 rounded accent-primary">`.
/// Never pre-ticked by the app (UC-5).
class ArchiveCheckbox extends StatelessWidget {
  const ArchiveCheckbox({super.key, required this.value, required this.onChanged, required this.child, this.semanticLabel});
  final bool value;
  final ValueChanged<bool> onChanged;
  final Widget child;
  final String? semanticLabel;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    return MergeSemantics(
      child: InkWell(
        onTap: () => onChanged(!value),
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
            SizedBox(
              width: 20,
              height: 20,
              child: Checkbox(
                value: value,
                onChanged: (v) => onChanged(v ?? false),
                materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                visualDensity: VisualDensity.compact,
                activeColor: cs.primary,
                checkColor: cs.onPrimary,
                side: BorderSide(color: cs.outline, width: 1.5),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                semanticLabel: semanticLabel,
              ),
            ),
            const SizedBox(width: 8),
            Expanded(child: child),
          ]),
        ),
      ),
    );
  }
}

/// Loads an image served by the API (relative `/v1/media/{id}` URLs need the bearer token).
class NetImage extends ConsumerWidget {
  const NetImage(this.url, {super.key, this.fit = BoxFit.cover, this.placeholder});
  final String? url;
  final BoxFit fit;
  final Color? placeholder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final bg = placeholder ?? context.cs.surfaceContainerHigh;
    final u = url;
    if (u == null || u.isEmpty) return ColoredBox(color: bg);
    final api = ref.watch(apiClientProvider);
    final full = u.startsWith('http') ? u : '${api.baseUrl}$u';
    return Image.network(
      full,
      headers: api.imageHeaders,
      fit: fit,
      gaplessPlayback: true,
      errorBuilder: (_, _, _) => ColoredBox(color: bg),
      frameBuilder: (_, child, frame, sync) => frame == null && !sync ? ColoredBox(color: bg) : child,
    );
  }
}

/// Profile picture: image when set, otherwise the initial on a container colour (as the archive's "ن" avatar).
class Avatar extends StatelessWidget {
  const Avatar({super.key, required this.author, this.size = 40, this.ring, this.ringWidth = 2, this.gap = 0, this.gapColor});
  final Author? author;
  final double size;

  /// Gradient ring (`p-0.5 bg-gradient-to-tr from-primary-container to-secondary`).
  final Gradient? ring;
  final double ringWidth, gap;
  final Color? gapColor;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final a = author;
    final initial = a == null ? '' : (a.name.isEmpty ? '' : String.fromCharCode(a.name.runes.first));
    Widget pic = ClipOval(
      child: (a?.avatarUrl ?? '').isNotEmpty
          ? NetImage(a!.avatarUrl, placeholder: cs.surfaceContainerHigh)
          : Container(
              color: cs.secondaryContainer,
              alignment: Alignment.center,
              child: Text(initial.toUpperCase(), style: tw(size * 0.42, size * 0.5, 700, cs.onSecondaryContainer)),
            ),
    );
    if (gap > 0) {
      pic = Container(padding: EdgeInsets.all(gap), decoration: BoxDecoration(color: gapColor ?? cs.surface, shape: BoxShape.circle), child: pic);
    }
    return Semantics(
      image: true,
      label: a == null ? null : 'تصویر ${a.name}',
      child: Container(
        width: size,
        height: size,
        padding: ring == null ? EdgeInsets.zero : EdgeInsets.all(ringWidth),
        decoration: ring == null ? null : BoxDecoration(gradient: ring, shape: BoxShape.circle),
        child: pic,
      ),
    );
  }
}

/// Gradient used by the archive's avatar/story rings: `bg-gradient-to-tr from-primary-container to-secondary`.
LinearGradient storyRing(ColorScheme cs, {bool via = false}) => LinearGradient(
      begin: AlignmentDirectional.bottomStart,
      end: AlignmentDirectional.topEnd,
      colors: via ? [cs.primaryContainer, cs.secondaryContainer, cs.secondary] : [cs.primaryContainer, cs.secondary],
    );

/// Bottom sheet listing actions (post / comment / user menus).
Future<T?> showActionSheet<T>(BuildContext context, List<(IconData, String, T, bool)> items, {String? title}) =>
    showModalBottomSheet<T>(
      context: context,
      showDragHandle: true,
      builder: (ctx) {
        final cs = ctx.cs;
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              if (title != null) Padding(padding: const EdgeInsets.only(bottom: 8), child: Text(title, style: tw(14, 20, 600, cs.onSurface))),
              for (final (icon, label, value, danger) in items)
                ListTile(
                  leading: Sy(icon, size: 22, color: danger ? cs.error : cs.onSurfaceVariant),
                  title: Text(label, style: tw(14, 20, 600, danger ? cs.error : cs.onSurface)),
                  onTap: () => Navigator.of(ctx).pop(value),
                ),
            ]),
          ),
        );
      },
    );

/// Confirmation dialog in archive colours.
Future<bool> confirm(BuildContext context, {required String title, required String message, required String action, bool danger = false}) async {
  final cs = context.cs;
  final ok = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      backgroundColor: cs.surfaceContainerLow,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      title: Text(title, style: tw(18, 24, 700, cs.onSurface)),
      content: Text(message, style: tw(14, 22, 400, cs.onSurfaceVariant)),
      actions: [
        TextButton(onPressed: () => Navigator.of(ctx).pop(false), child: Text('انصراف', style: tw(14, 20, 600, cs.onSurfaceVariant))),
        TextButton(onPressed: () => Navigator.of(ctx).pop(true), child: Text(action, style: tw(14, 20, 700, danger ? cs.error : cs.primary))),
      ],
    ),
  );
  return ok ?? false;
}
