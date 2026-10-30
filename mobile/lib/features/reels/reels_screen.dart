import 'package:flutter/material.dart';

import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/widgets/states.dart';
import '../home/home_screen.dart';

/// Reels keep their tab (UI lock); the player and upload are Phase 3 and are not activated.
class ReelsScreen extends StatelessWidget {
  const ReelsScreen({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
        backgroundColor: context.skin.page,
        appBar: const FeedHeader(title: 'ریلز'),
        body: const Center(child: FeatureNotYetView(icon: Sym.movie, title: 'ریلز و ویدیوهای کوتاه', phase: 'فاز ۳')),
      );
}
