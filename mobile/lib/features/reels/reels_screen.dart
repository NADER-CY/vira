import 'package:flutter/material.dart';

import '../../core/widgets/states.dart';

class ReelsScreen extends StatelessWidget {
  const ReelsScreen({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('ریلز')),
        body: const Center(child: FeatureNotYetView(icon: Icons.movie_outlined, title: 'ریلز و ویدیوهای کوتاه', phase: 'فاز ۳')),
      );
}
