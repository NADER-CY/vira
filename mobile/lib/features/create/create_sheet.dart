import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/states.dart';

/// Create menu (Phase 1 sheet, kept): post is active in Phase 2; reel and story (Phase 3) and live (Phase 8) keep
/// their entries, disabled, with the phase that delivers them.
Future<void> showCreateSheet(BuildContext context) => showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (ctx) {
        final cs = ctx.cs;
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.only(bottom: 16),
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              Text('ایجاد', style: tw(18, 24, 700, cs.onSurface)),
              const SizedBox(height: 8),
              ListTile(
                key: const Key('create-post'),
                leading: Sy(Sym.addPhotoAlternate, color: cs.primary),
                title: Text('پست', style: tw(14, 20, 600, cs.onSurface)),
                subtitle: Text('تصویر و متن', style: tw(12, 18, 400, cs.onSurfaceVariant)),
                onTap: () {
                  Navigator.of(ctx).pop();
                  context.push('/post/new');
                },
              ),
              for (final (icon, label, phase) in const [
                (Sym.movie, 'ریلز', 'فاز ۳'),
                (Sym.autoStories, 'استوری', 'فاز ۳'),
                (Sym.sensors, 'پخش زنده', 'فاز ۸'),
              ])
                ListTile(
                  leading: Sy(icon, color: cs.onSurfaceVariant.withValues(alpha: 0.6)),
                  title: Text(label, style: tw(14, 20, 600, cs.onSurface.withValues(alpha: 0.6))),
                  trailing: PhaseBadge(phase),
                  enabled: false,
                ),
            ]),
          ),
        );
      },
    );
