import 'package:flutter/material.dart';

import '../api/api_client.dart';
import '../theme/tokens.g.dart';

/// UI state catalogue (doc 09 §5): loading skeleton, empty, error with retry, feature disabled / locked.
class LoadingView extends StatelessWidget {
  const LoadingView({super.key, this.rows = 3});
  final int rows;

  @override
  Widget build(BuildContext context) {
    final c = Theme.of(context).colorScheme.surfaceContainerHighest;
    return Semantics(
      label: 'در حال بارگذاری',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          for (var i = 0; i < rows; i++)
            Container(
              height: i == 0 ? 24 : 16,
              margin: const EdgeInsetsDirectional.only(bottom: ViraTokens.space3, end: 0),
              decoration: BoxDecoration(color: c, borderRadius: BorderRadius.circular(ViraTokens.radiusSm)),
            ),
        ],
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
    final t = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: ViraTokens.space6, vertical: ViraTokens.space8),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircleAvatar(
            radius: 30,
            backgroundColor: t.colorScheme.primaryContainer,
            child: Icon(icon, color: t.colorScheme.onPrimaryContainer, size: 28),
          ),
          const SizedBox(height: ViraTokens.space4),
          Text(title, style: t.textTheme.titleMedium, textAlign: TextAlign.center),
          if (message != null) ...[
            const SizedBox(height: ViraTokens.space2),
            Text(message!, style: t.textTheme.bodyMedium?.copyWith(color: t.colorScheme.onSurfaceVariant), textAlign: TextAlign.center),
          ],
          if (action != null) ...[const SizedBox(height: ViraTokens.space4), action!],
        ],
      ),
    );
  }
}

class ErrorView extends StatelessWidget {
  const ErrorView({super.key, required this.error, this.onRetry});
  final Object error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final e = error is ApiException ? error as ApiException : null;
    return EmptyView(
      icon: Icons.cloud_off_outlined,
      title: e?.title ?? messageFor('internal_error'),
      message: e?.traceId != null ? 'کد پیگیری: \u2066${e!.traceId}\u2069' : null,
      action: onRetry == null ? null : OutlinedButton.icon(onPressed: onRetry, icon: const Icon(Icons.refresh), label: const Text('تلاش دوباره')),
    );
  }
}

/// A capability that is on the roadmap but not active yet. [locked] = regulated, behind an approval gate.
class FeatureNotYetView extends StatelessWidget {
  const FeatureNotYetView({super.key, required this.title, required this.phase, this.locked = false, this.icon});
  final String title;
  final String phase;
  final bool locked;
  final IconData? icon;

  @override
  Widget build(BuildContext context) => EmptyView(
        icon: icon ?? (locked ? Icons.lock_outline : Icons.hourglass_empty),
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
      padding: const EdgeInsets.symmetric(horizontal: ViraTokens.space2, vertical: 2),
      decoration: BoxDecoration(
        color: locked ? s.errorContainer : s.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(ViraTokens.radiusFull),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        if (locked) Icon(Icons.lock, size: 12, color: s.onErrorContainer),
        if (locked) const SizedBox(width: 4),
        Text(label, style: Theme.of(context).textTheme.labelSmall?.copyWith(color: locked ? s.onErrorContainer : s.onSurfaceVariant)),
      ]),
    );
  }
}
