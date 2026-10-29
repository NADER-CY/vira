import 'package:flutter/material.dart';

import '../../core/theme/tokens.g.dart';
import '../../core/widgets/states.dart';

class ExploreScreen extends StatelessWidget {
  const ExploreScreen({super.key});

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('کاوش')),
        body: ListView(padding: const EdgeInsets.all(ViraTokens.space4), children: const [
          TextField(
            enabled: false,
            decoration: InputDecoration(
              prefixIcon: Icon(Icons.search),
              hintText: 'جستجوی کاربران، محصولات و برچسب‌ها',
              border: OutlineInputBorder(borderRadius: BorderRadius.all(Radius.circular(ViraTokens.radiusMd))),
            ),
          ),
          SizedBox(height: ViraTokens.space4),
          FeatureNotYetView(icon: Icons.explore_outlined, title: 'کاوش و جستجو', phase: 'فاز ۲'),
        ]),
      );
}
