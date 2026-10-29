import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/models.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/states.dart';
import '../home/home_screen.dart';

/// Explore (`_26`, M-026): the header and the search field are active in Phase 2 for finding people; visual and
/// voice search, trend chips and the discovery grid are Phase 3.
class ExploreScreen extends ConsumerStatefulWidget {
  const ExploreScreen({super.key});

  @override
  ConsumerState<ExploreScreen> createState() => _ExploreScreenState();
}

class _ExploreScreenState extends ConsumerState<ExploreScreen> {
  final _q = TextEditingController();
  Timer? _debounce;
  Future<List<Author>>? _results;

  @override
  void dispose() {
    _debounce?.cancel();
    _q.dispose();
    super.dispose();
  }

  void _changed(String v) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () {
      final q = v.trim();
      setState(() => _results = q.length < 2 ? null : ref.read(repoProvider).searchUsers(q));
    });
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final sk = context.skin;
    return Scaffold(
      backgroundColor: sk.page,
      appBar: const FeedHeader(title: 'کاوش'),
      body: ListView(padding: const EdgeInsets.fromLTRB(12, 4, 12, 24), children: [
        Container(
          height: 44,
          decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(12)),
          child: Row(children: [
            const SizedBox(width: 6),
            Sy(Sym.search, size: 20, color: cs.onSurfaceVariant),
            const SizedBox(width: 6),
            Expanded(
              child: TextField(
                key: const Key('explore-search'),
                controller: _q,
                onChanged: _changed,
                style: tw(14, 20, 400, cs.onSurface),
                cursorColor: cs.primaryContainer,
                textInputAction: TextInputAction.search,
                decoration: InputDecoration(
                  isCollapsed: true,
                  border: InputBorder.none,
                  hintText: 'جستجوی کاربران با نام کاربری…',
                  hintStyle: tw(14, 20, 400, cs.onSurfaceVariant.withValues(alpha: 0.7)),
                ),
              ),
            ),
            RoundIconButton(icon: Sym.centerFocusStrong, label: 'جستجوی تصویری', size: 32, iconSize: 19, color: cs.primary, onTap: () => soon(context, 'جستجوی تصویری', 'فاز ۳')),
            const SizedBox(width: 4),
            RoundIconButton(icon: Sym.mic, label: 'جستجوی صوتی', size: 32, iconSize: 19, color: cs.onSurfaceVariant, onTap: () => soon(context, 'جستجوی صوتی', 'فاز ۳')),
            const SizedBox(width: 4),
          ]),
        ),
        const SizedBox(height: 12),
        if (_results == null)
          const FeatureNotYetView(icon: Sym.explore, title: 'ترندها و کاوش محتوا', phase: 'فاز ۳')
        else
          FutureBuilder<List<Author>>(
            future: _results,
            builder: (context, snap) {
              if (snap.connectionState != ConnectionState.done) return const LoadingView(rows: 3);
              if (snap.hasError) return ErrorView(error: snap.error!, onRetry: () => _changed(_q.text));
              final list = snap.data ?? const [];
              if (list.isEmpty) return const EmptyView(icon: Sym.personSearch, title: 'کاربری پیدا نشد');
              return Column(children: [
                for (final a in list)
                  ListTile(
                    contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                    leading: Avatar(author: a, size: 44),
                    title: Text(a.name, style: tw(14, 20, 600, cs.onSurface)),
                    subtitle: Text('@${isolate(a.handle)}', style: tw(12, 16, 400, cs.onSurfaceVariant)),
                    trailing: a.isPrivate ? Sy(Sym.lock, size: 16, color: cs.onSurfaceVariant) : null,
                    onTap: () => context.push('/u/${a.id}'),
                  ),
              ]);
            },
          ),
      ]),
    );
  }
}
