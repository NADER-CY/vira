import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/api/models.dart';
import 'core/auth/session.dart';
import 'core/providers.dart';
import 'core/theme/vira_theme.dart';
import 'features/activity/activity_screen.dart';
import 'features/auth/otp_screen.dart';
import 'features/auth/welcome_screen.dart';
import 'features/create/create_post_screen.dart';
import 'features/explore/explore_screen.dart';
import 'features/feed/comments_screen.dart';
import 'features/home/home_screen.dart';
import 'features/post/post_detail_screen.dart';
import 'features/profile/edit_profile_screen.dart';
import 'features/profile/follow_list_screen.dart';
import 'features/profile/profile_screen.dart';
import 'features/reels/reels_screen.dart';
import 'features/report/report_screen.dart';
import 'features/settings/account_screen.dart';
import 'features/settings/privacy_screen.dart';
import 'features/settings/security_screen.dart';
import 'features/settings/settings_screen.dart';
import 'features/shell/app_shell.dart';

/// Routes (doc 09 §4.1 deep links). Signed-out users only reach the sign-in flow; signed-in users skip it.
GoRouter buildRouter(Session session, {String initialLocation = '/home'}) => GoRouter(
      initialLocation: initialLocation,
      refreshListenable: session,
      redirect: (context, state) {
        final loc = state.matchedLocation;
        final inAuth = loc == '/welcome' || loc.startsWith('/auth');
        if (!session.signedIn && !inAuth) return '/welcome';
        if (session.signedIn && inAuth) return '/home';
        return null;
      },
      routes: [
        GoRoute(path: '/welcome', builder: (_, _) => const WelcomeScreen()),
        GoRoute(
          path: '/auth/verify',
          redirect: (_, state) => state.extra is OtpArgs ? null : '/welcome',
          builder: (_, state) => OtpScreen(args: state.extra! as OtpArgs),
        ),
        StatefulShellRoute.indexedStack(
          builder: (context, state, shell) => AppShell(shell: shell),
          branches: [
            StatefulShellBranch(routes: [GoRoute(path: '/home', builder: (_, _) => const HomeScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/explore', builder: (_, _) => const ExploreScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/reels', builder: (_, _) => const ReelsScreen())]),
            StatefulShellBranch(routes: [GoRoute(path: '/profile', builder: (_, _) => const ProfileScreen())]),
          ],
        ),
        GoRoute(path: '/profile/edit', builder: (_, _) => const EditProfileScreen()),
        GoRoute(path: '/u/:id', builder: (_, s) => ProfileScreen(userId: s.pathParameters['id'])),
        GoRoute(
          path: '/u/:id/followers',
          builder: (_, s) => FollowListScreen(userId: s.pathParameters['id']!, following: s.uri.queryParameters['tab'] == 'following'),
        ),
        GoRoute(path: '/post/new', builder: (_, _) => const CreatePostScreen()),
        GoRoute(path: '/post/:id', builder: (_, s) => PostDetailScreen(postId: s.pathParameters['id']!)),
        GoRoute(path: '/post/:id/edit', builder: (_, s) => CreatePostScreen(editing: s.extra is Post ? s.extra! as Post : null)),
        GoRoute(path: '/post/:id/comments', builder: (_, s) => CommentsScreen(postId: s.pathParameters['id']!, post: s.extra is Post ? s.extra! as Post : null)),
        GoRoute(path: '/activity', builder: (_, _) => const ActivityScreen()),
        GoRoute(path: '/settings', builder: (_, _) => const SettingsScreen()),
        GoRoute(path: '/settings/privacy', builder: (_, _) => const PrivacyScreen()),
        GoRoute(path: '/settings/security', builder: (_, _) => const SecurityScreen()),
        GoRoute(path: '/settings/account', builder: (_, _) => const AccountScreen()),
        GoRoute(
          path: '/report',
          redirect: (_, state) => state.extra is ReportArgs ? null : '/home',
          builder: (_, s) => ReportScreen(args: s.extra! as ReportArgs),
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
  late final GoRouter _router = buildRouter(ref.read(sessionProvider), initialLocation: widget.initialLocation);

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
