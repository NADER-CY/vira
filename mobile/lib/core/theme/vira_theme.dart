import 'package:flutter/material.dart';

import 'archive_tokens.g.dart';
import 'tokens.g.dart';

/// Material 3 themes built 1:1 from the colour sets of the original UI archive (UI LOCK, Phase 2).
///
/// Every ColorScheme role is the archive's Tailwind token of the same name (light: 79 screens, background #faf9ff /
/// primary #ba0038; dark: 112 screens, background #0f131c / primary #ffb2ba). The Phase 1 approximations
/// (#0B0F17 / #121826 …) are no longer used by the app — see docs/progress/P2.md §UI.
abstract final class ViraTheme {
  static ThemeData light() => _build(Brightness.light);
  static ThemeData dark() => _build(Brightness.dark);

  static ColorScheme scheme(Brightness b) => b == Brightness.dark
      ? const ColorScheme(
          brightness: Brightness.dark,
          primary: ArchiveDark.primary,
          onPrimary: ArchiveDark.onPrimary,
          primaryContainer: ArchiveDark.primaryContainer,
          onPrimaryContainer: ArchiveDark.onPrimaryContainer,
          primaryFixed: ArchiveDark.primaryFixed,
          primaryFixedDim: ArchiveDark.primaryFixedDim,
          onPrimaryFixed: ArchiveDark.onPrimaryFixed,
          onPrimaryFixedVariant: ArchiveDark.onPrimaryFixedVariant,
          secondary: ArchiveDark.secondary,
          onSecondary: ArchiveDark.onSecondary,
          secondaryContainer: ArchiveDark.secondaryContainer,
          onSecondaryContainer: ArchiveDark.onSecondaryContainer,
          secondaryFixed: ArchiveDark.secondaryFixed,
          secondaryFixedDim: ArchiveDark.secondaryFixedDim,
          onSecondaryFixed: ArchiveDark.onSecondaryFixed,
          onSecondaryFixedVariant: ArchiveDark.onSecondaryFixedVariant,
          tertiary: ArchiveDark.tertiary,
          onTertiary: ArchiveDark.onTertiary,
          tertiaryContainer: ArchiveDark.tertiaryContainer,
          onTertiaryContainer: ArchiveDark.onTertiaryContainer,
          tertiaryFixed: ArchiveDark.tertiaryFixed,
          tertiaryFixedDim: ArchiveDark.tertiaryFixedDim,
          onTertiaryFixed: ArchiveDark.onTertiaryFixed,
          onTertiaryFixedVariant: ArchiveDark.onTertiaryFixedVariant,
          error: ArchiveDark.error,
          onError: ArchiveDark.onError,
          errorContainer: ArchiveDark.errorContainer,
          onErrorContainer: ArchiveDark.onErrorContainer,
          surface: ArchiveDark.surface,
          onSurface: ArchiveDark.onSurface,
          surfaceDim: ArchiveDark.surfaceDim,
          surfaceBright: ArchiveDark.surfaceBright,
          surfaceContainerLowest: ArchiveDark.surfaceContainerLowest,
          surfaceContainerLow: ArchiveDark.surfaceContainerLow,
          surfaceContainer: ArchiveDark.surfaceContainer,
          surfaceContainerHigh: ArchiveDark.surfaceContainerHigh,
          surfaceContainerHighest: ArchiveDark.surfaceContainerHighest,
          onSurfaceVariant: ArchiveDark.onSurfaceVariant,
          outline: ArchiveDark.outline,
          outlineVariant: ArchiveDark.outlineVariant,
          shadow: Colors.black,
          scrim: Colors.black,
          inverseSurface: ArchiveDark.inverseSurface,
          onInverseSurface: ArchiveDark.inverseOnSurface,
          inversePrimary: ArchiveDark.inversePrimary,
          surfaceTint: ArchiveDark.surfaceTint,
        )
      : const ColorScheme(
          brightness: Brightness.light,
          primary: ArchiveLight.primary,
          onPrimary: ArchiveLight.onPrimary,
          primaryContainer: ArchiveLight.primaryContainer,
          onPrimaryContainer: ArchiveLight.onPrimaryContainer,
          primaryFixed: ArchiveLight.primaryFixed,
          primaryFixedDim: ArchiveLight.primaryFixedDim,
          onPrimaryFixed: ArchiveLight.onPrimaryFixed,
          onPrimaryFixedVariant: ArchiveLight.onPrimaryFixedVariant,
          secondary: ArchiveLight.secondary,
          onSecondary: ArchiveLight.onSecondary,
          secondaryContainer: ArchiveLight.secondaryContainer,
          onSecondaryContainer: ArchiveLight.onSecondaryContainer,
          secondaryFixed: ArchiveLight.secondaryFixed,
          secondaryFixedDim: ArchiveLight.secondaryFixedDim,
          onSecondaryFixed: ArchiveLight.onSecondaryFixed,
          onSecondaryFixedVariant: ArchiveLight.onSecondaryFixedVariant,
          tertiary: ArchiveLight.tertiary,
          onTertiary: ArchiveLight.onTertiary,
          tertiaryContainer: ArchiveLight.tertiaryContainer,
          onTertiaryContainer: ArchiveLight.onTertiaryContainer,
          tertiaryFixed: ArchiveLight.tertiaryFixed,
          tertiaryFixedDim: ArchiveLight.tertiaryFixedDim,
          onTertiaryFixed: ArchiveLight.onTertiaryFixed,
          onTertiaryFixedVariant: ArchiveLight.onTertiaryFixedVariant,
          error: ArchiveLight.error,
          onError: ArchiveLight.onError,
          errorContainer: ArchiveLight.errorContainer,
          onErrorContainer: ArchiveLight.onErrorContainer,
          surface: ArchiveLight.surface,
          onSurface: ArchiveLight.onSurface,
          surfaceDim: ArchiveLight.surfaceDim,
          surfaceBright: ArchiveLight.surfaceBright,
          surfaceContainerLowest: ArchiveLight.surfaceContainerLowest,
          surfaceContainerLow: ArchiveLight.surfaceContainerLow,
          surfaceContainer: ArchiveLight.surfaceContainer,
          surfaceContainerHigh: ArchiveLight.surfaceContainerHigh,
          surfaceContainerHighest: ArchiveLight.surfaceContainerHighest,
          onSurfaceVariant: ArchiveLight.onSurfaceVariant,
          outline: ArchiveLight.outline,
          outlineVariant: ArchiveLight.outlineVariant,
          shadow: Colors.black,
          scrim: Colors.black,
          inverseSurface: ArchiveLight.inverseSurface,
          onInverseSurface: ArchiveLight.inverseOnSurface,
          inversePrimary: ArchiveLight.inversePrimary,
          surfaceTint: ArchiveLight.surfaceTint,
        );

  static ThemeData _build(Brightness b) {
    final s = scheme(b);
    final text = ViraType.textTheme(s.onSurface);
    return ThemeData(
      useMaterial3: true,
      brightness: b,
      colorScheme: s,
      scaffoldBackgroundColor: s.surface,
      canvasColor: s.surface,
      fontFamily: ViraTokens.fontFamily,
      textTheme: text,
      extensions: [ViraColors.of(b), FeedSkin.of(b)],
      appBarTheme: AppBarTheme(
        backgroundColor: s.surface,
        foregroundColor: s.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0,
        titleTextStyle: ViraType.headlineSm.copyWith(color: s.onSurface),
      ),
      dividerTheme: DividerThemeData(color: s.outlineVariant, space: 1, thickness: 1),
      snackBarTheme: SnackBarThemeData(
        backgroundColor: s.inverseSurface,
        contentTextStyle: ViraType.bodyMd.copyWith(color: s.onInverseSurface),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(ArchiveRadius.xl)),
      ),
      bottomSheetTheme: BottomSheetThemeData(
        backgroundColor: s.surfaceContainerLow,
        modalBackgroundColor: s.surfaceContainerLow,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(ArchiveRadius.x3l))),
      ),
      textSelectionTheme: TextSelectionThemeData(cursorColor: s.primaryContainer, selectionColor: s.primaryContainer.withValues(alpha: 0.3)),
      progressIndicatorTheme: ProgressIndicatorThemeData(color: s.primaryContainer),
    );
  }
}

/// The archive type scale (Tailwind `text-*` + `font-*` pairs). Persian text is set in Vazirmatn everywhere (ADR-014).
abstract final class ViraType {
  static TextStyle _s(({double size, double height, int weight}) t) => TextStyle(
        fontFamily: ViraTokens.fontFamily,
        fontSize: t.size,
        height: t.height,
        fontWeight: FontWeight.values[(t.weight ~/ 100) - 1],
        leadingDistribution: TextLeadingDistribution.even,
      );

  static final headlineXl = _s(ArchiveType.headlineXl);
  static final headlineLg = _s(ArchiveType.headlineLg);
  static final headlineMd = _s(ArchiveType.headlineMd);
  static final headlineSm = _s(ArchiveType.headlineSm);
  static final priceDisplay = _s(ArchiveType.priceDisplay);
  static final bodyLg = _s(ArchiveType.bodyLg);
  static final bodyMd = _s(ArchiveType.bodyMd);
  static final bodySm = _s(ArchiveType.bodySm);
  static final labelLg = _s(ArchiveType.labelLg);
  static final labelMd = _s(ArchiveType.labelMd);
  static final labelSm = _s(ArchiveType.labelSm);

  /// Tailwind arbitrary sizes used by the archive (`text-[10px]`, `text-[11px]` …): body line height 1.5.
  static TextStyle px(double size, {FontWeight weight = FontWeight.w400}) => TextStyle(
        fontFamily: ViraTokens.fontFamily,
        fontSize: size,
        height: 1.5,
        fontWeight: weight,
        leadingDistribution: TextLeadingDistribution.even,
      );

  static TextTheme textTheme(Color color) => TextTheme(
        displaySmall: headlineXl.copyWith(color: color),
        headlineLarge: headlineLg.copyWith(color: color),
        headlineMedium: headlineMd.copyWith(color: color),
        headlineSmall: headlineSm.copyWith(color: color),
        titleLarge: headlineMd.copyWith(color: color),
        titleMedium: headlineSm.copyWith(color: color),
        titleSmall: labelLg.copyWith(color: color),
        bodyLarge: bodyLg.copyWith(color: color),
        bodyMedium: bodyMd.copyWith(color: color),
        bodySmall: bodySm.copyWith(color: color),
        labelLarge: labelLg.copyWith(color: color),
        labelMedium: labelMd.copyWith(color: color),
        labelSmall: labelSm.copyWith(color: color),
      );
}

/// Colours of the home shell (feed, header, bottom bar). Dark = `_33` (archive tokens). Light = `home_feed_light`,
/// whose markup sets its colours inline (#FF2D65, #151923, #667085, #E6E8ED …) instead of the light token set;
/// elements it leaves on tokens use the same inline values (conflict OD-P2-02 in docs/progress/P2.md).
@immutable
class FeedSkin extends ThemeExtension<FeedSkin> {
  const FeedSkin({
    required this.page,
    required this.header,
    required this.headerBorder,
    required this.text,
    required this.muted,
    required this.accent,
    required this.onAccent,
    required this.badge,
    required this.card,
    required this.cardBorder,
    required this.cardShadow,
    required this.cardGap,
    required this.media,
    required this.segment,
    required this.chip,
    required this.chipBorder,
    required this.liveBg,
    required this.liveFg,
    required this.liveBorder,
    required this.shopBg,
    required this.shopGradient,
    required this.shopFg,
    required this.createBg,
    required this.createGradient,
    required this.createFg,
    required this.navBg,
    required this.navBorder,
    required this.navShadow,
    required this.navActive,
    required this.navInactive,
    required this.storyDivider,
    required this.endCard,
    required this.endCardBorder,
    required this.panel,
  });

  factory FeedSkin.of(Brightness b) => b == Brightness.dark ? dark : light;

  static const dark = FeedSkin(
    page: ArchiveDark.surface,
    header: Color(0xD90F131C), // bg-surface/85
    headerBorder: null,
    text: ArchiveDark.onSurface,
    muted: ArchiveDark.onSurfaceVariant,
    accent: ArchiveDark.primary,
    onAccent: ArchiveDark.onPrimaryContainer,
    badge: ArchiveDark.primaryContainer,
    card: ArchiveDark.surfaceContainer,
    cardBorder: null,
    cardShadow: [BoxShadow(color: Color(0x40000000), blurRadius: 25, offset: Offset(0, 20), spreadRadius: -5)],
    cardGap: 8,
    media: ArchiveDark.surfaceContainerLow,
    segment: ArchiveDark.surfaceContainerLow,
    chip: ArchiveDark.surfaceContainer,
    chipBorder: null,
    liveBg: ArchiveDark.surfaceContainer,
    liveFg: ArchiveDark.primary,
    liveBorder: null,
    shopBg: Color(0x33FF4F74), // bg-primary-container/20
    shopGradient: null,
    shopFg: ArchiveDark.primaryContainer,
    createBg: ArchiveDark.surfaceContainerHigh,
    createGradient: null,
    createFg: ArchiveDark.onSurface,
    navBg: Color(0xE60F131C), // bg-surface/90
    navBorder: null,
    navShadow: [BoxShadow(color: Color(0x66000000), blurRadius: 16, offset: Offset(0, -1))],
    navActive: ArchiveDark.primary,
    navInactive: ArchiveDark.onSurfaceVariant,
    storyDivider: null,
    endCard: ArchiveDark.surfaceContainer,
    endCardBorder: null,
    panel: ArchiveDark.surfaceContainerHigh,
  );

  static const light = FeedSkin(
    page: Color(0xFFFFFFFF),
    header: Color(0xF2FFFFFF), // rgba(255,255,255,0.95)
    headerBorder: Color(0xFFE6E8ED),
    text: Color(0xFF151923),
    muted: Color(0xFF667085),
    accent: Color(0xFFFF2D65),
    onAccent: Color(0xFFFFFFFF),
    badge: Color(0xFFFF2D65),
    card: Color(0xFFFFFFFF),
    cardBorder: Color(0xFFE6E8ED),
    cardShadow: [BoxShadow(color: Color(0x0A000000), blurRadius: 12, offset: Offset(0, 2))],
    cardGap: 12,
    media: Color(0xFFF2F4F7),
    segment: Color(0xFFF2F4F7),
    chip: Color(0xFFF7F8FA),
    chipBorder: Color(0xFFE6E8ED),
    liveBg: Color(0xFFFFF0F4),
    liveFg: Color(0xFFFF2D65),
    liveBorder: Color(0xFFFFE0E8),
    shopBg: null,
    shopGradient: [Color(0xFFFF2D65), Color(0xFFBA0038)],
    shopFg: Color(0xFFFFFFFF),
    createBg: null,
    createGradient: [Color(0xFFFF2D65), Color(0xFF7B61FF)],
    createFg: Color(0xFFFFFFFF),
    navBg: Color(0xFAFFFFFF), // rgba(255,255,255,0.98)
    navBorder: Color(0xFFE6E8ED),
    navShadow: [BoxShadow(color: Color(0x08000000), blurRadius: 12, offset: Offset(0, -2))],
    navActive: Color(0xFFFF2D65),
    navInactive: Color(0xFF667085),
    storyDivider: Color(0xFFF0F2F5),
    endCard: Color(0xFFF7F8FA),
    endCardBorder: Color(0xFFE6E8ED),
    panel: Color(0xFFF7F8FA),
  );

  final Color page, header, text, muted, accent, onAccent, badge, card, media, segment, chip, liveBg, liveFg, shopFg;
  final Color createFg, navBg, navActive, navInactive, endCard, panel;
  final Color? headerBorder, cardBorder, chipBorder, liveBorder, shopBg, createBg, navBorder, storyDivider, endCardBorder;
  final List<Color>? shopGradient, createGradient;
  final List<BoxShadow> cardShadow, navShadow;
  final double cardGap;

  @override
  FeedSkin copyWith() => this;

  @override
  FeedSkin lerp(FeedSkin? other, double t) => t < 0.5 || other == null ? this : other;
}

/// Semantic colours Material's ColorScheme does not cover (doc 09 §2), kept from Phase 1 for status chips.
@immutable
class ViraColors extends ThemeExtension<ViraColors> {
  const ViraColors({required this.success, required this.warning, required this.info});

  factory ViraColors.of(Brightness b) => b == Brightness.dark
      ? const ViraColors(success: ViraDarkColors.success, warning: ViraDarkColors.warning, info: ViraDarkColors.info)
      : const ViraColors(success: ViraLightColors.success, warning: ViraLightColors.warning, info: ViraLightColors.info);

  final Color success, warning, info;

  @override
  ViraColors copyWith() => this;

  @override
  ViraColors lerp(ViraColors? other, double t) => t < 0.5 || other == null ? this : other;
}

extension ViraThemeX on BuildContext {
  ColorScheme get cs => Theme.of(this).colorScheme;
  ViraColors get vira => Theme.of(this).extension<ViraColors>()!;
  FeedSkin get skin => Theme.of(this).extension<FeedSkin>()!;
  bool get isDark => Theme.of(this).brightness == Brightness.dark;
}
