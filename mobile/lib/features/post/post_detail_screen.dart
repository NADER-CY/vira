import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/models.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/widgets/states.dart';
import '../feed/post_card.dart';

/// A single post (deep link /post/{id}): the feed's post card (`_33` / `home_feed_light`) on its own page.
class PostDetailScreen extends ConsumerStatefulWidget {
  const PostDetailScreen({super.key, required this.postId});
  final String postId;

  @override
  ConsumerState<PostDetailScreen> createState() => _PostDetailScreenState();
}

class _PostDetailScreenState extends ConsumerState<PostDetailScreen> {
  Post? _post;
  Object? _error;
  bool _deleted = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final p = await ref.read(repoProvider).post(widget.postId);
      if (mounted) setState(() => _post = p);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    final p = _post;
    return Scaffold(
      backgroundColor: sk.page,
      appBar: ArchiveHeader(title: p == null ? 'پست' : 'پست ${isolateHandle(p.author.handle)}', horizontal: 12),
      body: _deleted
          ? const Center(child: EmptyView(icon: Sym.delete, title: 'این پست حذف شد'))
          : p == null
              ? (_error == null ? const LoadingView(rows: 4) : Center(child: ErrorView(error: _error!, onRetry: _load)))
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(padding: const EdgeInsets.only(top: 8, bottom: 24), children: [
                    PostCard(
                      post: p,
                      detail: true,
                      onChanged: (n) => _post = n,
                      onDeleted: () {
                        setState(() => _deleted = true);
                        if (context.canPop()) context.pop();
                      },
                    ),
                  ]),
                ),
    );
  }
}

String isolateHandle(String h) => '\u2068@$h\u2069';
