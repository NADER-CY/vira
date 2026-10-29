import 'package:flutter/material.dart';

import '../../core/theme/tokens.g.dart';
import '../../core/widgets/states.dart';

/// Create menu (post, reel, story, live). Each entry is labelled with the phase that builds it.
Future<void> showCreateSheet(BuildContext context) => showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.only(bottom: ViraTokens.space4),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Text('ایجاد', style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: ViraTokens.space2),
            for (final (icon, label, phase) in const [
              (Icons.photo_library_outlined, 'پست', 'فاز ۲'),
              (Icons.movie_creation_outlined, 'ریلز', 'فاز ۳'),
              (Icons.amp_stories_outlined, 'استوری', 'فاز ۳'),
              (Icons.sensors, 'پخش زنده', 'فاز ۸'),
            ])
              ListTile(
                leading: Icon(icon),
                title: Text(label),
                trailing: PhaseBadge(phase),
                enabled: false,
              ),
          ]),
        ),
      ),
    );
