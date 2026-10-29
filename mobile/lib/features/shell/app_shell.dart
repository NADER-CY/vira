import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/widgets/brand.dart';
import '../create/create_sheet.dart';

/// Bottom navigation: Home · Explore · ＋ · Reels · Profile — the most frequent pattern in the archive
/// (doc 09 §4, D-28 provisional). In RTL the first destination (Home) is on the right.
class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.shell});
  final StatefulNavigationShell shell;

  // Branch index → bar index (the create button sits in the middle and is not a branch).
  static const _barForBranch = [0, 1, 3, 4];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: shell,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _barForBranch[shell.currentIndex],
        onDestinationSelected: (i) {
          if (i == 2) {
            showCreateSheet(context);
            return;
          }
          final branch = _barForBranch.indexOf(i);
          shell.goBranch(branch, initialLocation: branch == shell.currentIndex);
        },
        destinations: [
          const NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'خانه'),
          const NavigationDestination(icon: Icon(Icons.explore_outlined), selectedIcon: Icon(Icons.explore), label: 'کاوش'),
          NavigationDestination(icon: CreateButton(onPressed: () => showCreateSheet(context)), label: ''),
          const NavigationDestination(icon: Icon(Icons.movie_outlined), selectedIcon: Icon(Icons.movie), label: 'ریلز'),
          const NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: 'پروفایل'),
        ],
      ),
    );
  }
}
