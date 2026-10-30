import 'package:flutter/material.dart';

import '../api/api_client.dart';
import '../theme/symbols.g.dart';
import '../ui/kit.dart';

/// UI state catalogue (doc 09 §5): loading, empty, error with retry, feature not yet active. Drawn with the
/// archive's empty-card pattern (icon disc, title, text, pill action).
String errorTitle(Object e) => e is ApiException ? e.title : messageFor('internal_error');

String? errorDetail(Object e) => e is ApiException && e.traceId != null ? 'کد پیگیری: \u2066${e.traceId}\u2069' : null;

class LoadingView extends StatelessWidget {
  const LoadingView({super.key, this.rows = 3});
  final int rows;

  @override
  Widget build(BuildContext context) {
    final c = Theme.of(context).colorScheme.surfaceContainerHigh;
    return Semantics(
      label: 'در حال بارگذاری',
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            for (var i = 0; i < rows; i++)
              Container(
                height: i == 0 ? 24 : 16,
                margin: const EdgeInsets.only(bottom: 12),
                decoration: BoxDecoration(color: c, borderRadius: BorderRadius.circular(8)),
              ),
          ],
        ),
      ),
    );
  }
}

class EmptyView extends StatelessWidget {
  const EmptyView({super.key, required this.icon, required this.title, this.message, this.action});
  final IconData icon;
  final String title;
  final String? message;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(color: cs.primaryContainer.withValues(alpha: 0.2), shape: BoxShape.circle),
            alignment: Alignment.center,
            child: Sy(icon, size: 26, color: cs.primary),
          ),
          const SizedBox(height: 8),
          Text(title, style: tw(18, 24, 700, cs.onSurface), textAlign: TextAlign.center),
          if (message != null) ...[
            const SizedBox(height: 4),
            Text(message!, style: tw(12, 18, 400, cs.onSurfaceVariant), textAlign: TextAlign.center),
          ],
          if (action != null) ...[const SizedBox(height: 14), action!],
        ],
      ),
    );
  }
}

/// Pill action used inside empty / error states (`rounded-full bg-surface-container-high px-5 py-2`).
class PillButton extends StatelessWidget {
  const PillButton({super.key, required this.label, required this.icon, required this.onTap});
  final String label;
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Material(
      color: cs.surfaceContainerHigh,
      borderRadius: BorderRadius.circular(999),
      child: InkWell(
        borderRadius: BorderRadius.circular(999),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            Sy(icon, size: 18, color: cs.onSurface),
            const SizedBox(width: 4),
            Text(label, style: tw(12, 16, 600, cs.onSurface)),
          ]),
        ),
      ),
    );
  }
}

class ErrorView extends StatelessWidget {
  const ErrorView({super.key, required this.error, this.onRetry});
  final Object error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) => EmptyView(
        icon: Sym.cloudOff,
        title: errorTitle(error),
        message: errorDetail(error),
        action: onRetry == null ? null : PillButton(label: 'تلاش دوباره', icon: Sym.refresh, onTap: onRetry!),
      );
}

/// A capability that keeps its place from the archive but is not active yet. [locked] = regulated, behind a gate.
class FeatureNotYetView extends StatelessWidget {
  const FeatureNotYetView({super.key, required this.title, required this.phase, this.locked = false, this.icon});
  final String title;
  final String phase;
  final bool locked;
  final IconData? icon;

  @override
  Widget build(BuildContext context) => EmptyView(
        icon: icon ?? (locked ? Sym.lock : Sym.hourglassEmpty),
        title: title,
        message: locked
            ? 'این قابلیت تا عبور از دروازهٔ $phase (تأیید حقوقی، امنیتی و تأمین‌کننده) قفل است.'
            : 'این بخش در $phase فعال می‌شود.',
      );
}

/// Small pill showing the phase or gate of a not-yet-active entry.
class PhaseBadge extends StatelessWidget {
  const PhaseBadge(this.label, {super.key, this.locked = false});
  final String label;
  final bool locked;

  @override
  Widget build(BuildContext context) {
    final s = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: locked ? s.errorContainer : s.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        if (locked) Sy(Sym.lock, size: 12, color: s.onErrorContainer),
        if (locked) const SizedBox(width: 4),
        Text(label, style: tw(11, 14, 500, locked ? s.onErrorContainer : s.onSurfaceVariant)),
      ]),
    );
  }
}
