import 'package:flutter/material.dart';

import 'tokens.g.dart';

/// Builds Material 3 themes from the generated VIRA tokens (doc 09 §2; DS-01 single seed, DS-02 committed tokens).
abstract final class ViraTheme {
  static ThemeData light() => _build(Brightness.light);
  static ThemeData dark() => _build(Brightness.dark);

  static ThemeData _build(Brightness b) {
    final dark = b == Brightness.dark;
    final scheme = ColorScheme(
      brightness: b,
      primary: dark ? ViraDarkColors.primary : ViraLightColors.primary,
      onPrimary: dark ? ViraDarkColors.onPrimary : ViraLightColors.onPrimary,
      primaryContainer: dark ? ViraDarkColors.primaryContainer : ViraLightColors.primaryContainer,
      onPrimaryContainer: dark ? ViraDarkColors.onPrimaryContainer : ViraLightColors.onPrimaryContainer,
      secondary: dark ? ViraDarkColors.secondary : ViraLightColors.secondary,
      onSecondary: dark ? ViraDarkColors.onSecondary : ViraLightColors.onSecondary,
      tertiary: dark ? ViraDarkColors.accent : ViraLightColors.accent,
      onTertiary: Colors.white,
      error: dark ? ViraDarkColors.danger : ViraLightColors.danger,
      onError: dark ? ViraDarkColors.background : Colors.white,
      errorContainer: dark ? ViraDarkColors.dangerContainer : ViraLightColors.dangerContainer,
      onErrorContainer: dark ? ViraDarkColors.danger : ViraLightColors.danger,
      surface: dark ? ViraDarkColors.surface : ViraLightColors.surface,
      onSurface: dark ? ViraDarkColors.onSurface : ViraLightColors.onSurface,
      surfaceContainerHighest: dark ? ViraDarkColors.surfaceVariant : ViraLightColors.surfaceVariant,
      onSurfaceVariant: dark ? ViraDarkColors.onSurfaceVariant : ViraLightColors.onSurfaceVariant,
      outline: dark ? ViraDarkColors.outlineStrong : ViraLightColors.outlineStrong,
      outlineVariant: dark ? ViraDarkColors.outline : ViraLightColors.outline,
    );
    final background = dark ? ViraDarkColors.background : ViraLightColors.background;
    final text = _textTheme(scheme.onSurface);
    return ThemeData(
      useMaterial3: true,
      brightness: b,
      colorScheme: scheme,
      scaffoldBackgroundColor: background,
      fontFamily: ViraTokens.fontFamily,
      textTheme: text,
      extensions: [ViraColors.of(b)],
      appBarTheme: AppBarTheme(
        backgroundColor: background,
        foregroundColor: scheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0,
        titleTextStyle: text.titleLarge,
      ),
      cardTheme: CardThemeData(
        color: scheme.surface,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(ViraTokens.radiusLg),
          side: BorderSide(color: scheme.outlineVariant),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: scheme.surface,
        indicatorColor: scheme.primaryContainer,
        height: 68,
        labelTextStyle: WidgetStatePropertyAll(text.labelMedium),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size(ViraTokens.minTarget, ViraTokens.minTarget),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(ViraTokens.radiusMd)),
          textStyle: text.labelLarge,
        ),
      ),
      chipTheme: ChipThemeData(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(ViraTokens.radiusFull)),
        labelStyle: text.labelMedium,
      ),
      dividerTheme: DividerThemeData(color: scheme.outlineVariant, space: 1),
    );
  }

  static TextTheme _textTheme(Color color) {
    TextStyle s(({double size, double height, int weight}) t) => TextStyle(
          fontFamily: ViraTokens.fontFamily,
          fontSize: t.size,
          height: t.height,
          fontWeight: FontWeight.values[(t.weight ~/ 100) - 1],
          color: color,
        );
    return TextTheme(
      displaySmall: s(ViraTypeScale.displaySm),
      titleLarge: s(ViraTypeScale.titleLg),
      titleMedium: s(ViraTypeScale.titleMd),
      bodyLarge: s(ViraTypeScale.bodyLg),
      bodyMedium: s(ViraTypeScale.bodyMd),
      labelLarge: s(ViraTypeScale.labelMd),
      labelMedium: s(ViraTypeScale.labelMd),
      bodySmall: s(ViraTypeScale.caption),
      labelSmall: s(ViraTypeScale.caption),
    );
  }
}

/// Commerce semantic colours that Material's ColorScheme does not cover (doc 09 §2).
@immutable
class ViraColors extends ThemeExtension<ViraColors> {
  const ViraColors({
    required this.success,
    required this.warning,
    required this.info,
    required this.price,
    required this.priceDiscount,
    required this.live,
    required this.verified,
    required this.sponsored,
    required this.gradientBrand,
  });

  factory ViraColors.of(Brightness b) => b == Brightness.dark
      ? const ViraColors(
          success: ViraDarkColors.success,
          warning: ViraDarkColors.warning,
          info: ViraDarkColors.info,
          price: ViraDarkColors.price,
          priceDiscount: ViraDarkColors.priceDiscount,
          live: ViraDarkColors.live,
          verified: ViraDarkColors.verified,
          sponsored: ViraDarkColors.sponsored,
          gradientBrand: ViraTokens.gradientBrand,
        )
      : const ViraColors(
          success: ViraLightColors.success,
          warning: ViraLightColors.warning,
          info: ViraLightColors.info,
          price: ViraLightColors.price,
          priceDiscount: ViraLightColors.priceDiscount,
          live: ViraLightColors.live,
          verified: ViraLightColors.verified,
          sponsored: ViraLightColors.sponsored,
          gradientBrand: ViraTokens.gradientBrand,
        );

  final Color success, warning, info, price, priceDiscount, live, verified, sponsored;
  final List<Color> gradientBrand;

  @override
  ViraColors copyWith() => this;

  @override
  ViraColors lerp(ViraColors? other, double t) => t < 0.5 || other == null ? this : other;
}

extension ViraThemeX on BuildContext {
  ViraColors get vira => Theme.of(this).extension<ViraColors>()!;
}
