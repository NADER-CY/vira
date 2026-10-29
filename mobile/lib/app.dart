import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/providers.dart';
import 'core/theme/vira_theme.dart';
import 'features/explore/explore_screen.dart';
import 'features/home/home_screen.dart';
import 'features/profile/profile_screen.dart';
import 'features/reels/reels_screen.dart';
import 'features/shell/app_shell.dart';

/// Deep links (doc 09 §4.1) are added per phase; Phase 1 registers the tab roots.
GoRouter buildRouter({String initialLocation = '/home'}) => GoRouter(
      initialLocation: initialLocation,
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (context, state, shell) => AppShell(shell: shell),
          branches: [
            StatefulShellBranch(routes: [GoRoute(path: '/home', builder: (_, _) => const HomeScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/explore', builder: (_, _) => const ExploreScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/reels', builder: (_, _) => const ReelsScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/profile', builder: (_, _) => const ProfileScreen())]),
          ],
        ),
      ],
    );

class ViraApp extends ConsumerStatefulWidget {
  const ViraApp({super.key, this.initialLocation = '/home'});
  final String initialLocation;

  @override
  ConsumerState<ViraApp> createState() => _ViraAppState();
}

class _ViraAppState extends ConsumerState<ViraApp> {
  late final GoRouter _router = buildRouter(initialLocation: widget.initialLocation);

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => MaterialApp.router(
        title: 'ویرا',
        debugShowCheckedModeBanner: false,
        // Persian-first: the whole app is right-to-left (doc 09 §3).
        locale: const Locale('fa', 'IR'),
        supportedLocales: const [Locale('fa', 'IR')],
        localizationsDelegates: GlobalMaterialLocalizations.delegates,
        theme: ViraTheme.light(),
        darkTheme: ViraTheme.dark(),
        themeMode: ref.watch(themeModeProvider),
        routerConfig: _router,
      );
}
