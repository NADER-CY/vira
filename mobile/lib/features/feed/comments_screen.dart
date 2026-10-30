import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api/api_client.dart';
import '../../core/api/models.dart';
import '../../core/api/repository.dart';
import '../../core/format/persian.dart';
import '../../core/providers.dart';
import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../../core/ui/paged.dart';
import '../../core/widgets/states.dart';
import '../report/report_screen.dart';

/// Comments. Reference: `comments` (M-242, dark; stale label "Shopping Cart" corrected): dimmed post image,
/// rounded sheet with handle, title + count, sort and close, comment cards, quick reactions, composer.
/// Phase 2: list (paginated), add (idempotent), delete own / post owner's, report others. The seller's pinned
/// comment with product card is Phase 4; comment likes and threaded replies are later phases.
class CommentsScreen extends ConsumerStatefulWidget {
  const CommentsScreen({super.key, required this.postId, this.post});
  final String postId;
  final Post? post;

  @override
  ConsumerState<CommentsScreen> createState() => _CommentsScreenState();
}

class _CommentsScreenState extends ConsumerState<CommentsScreen> {
  late final Paged<Comment> _list = Paged<Comment>((c) => ref.read(repoProvider).comments(widget.postId, cursor: c))..more();
  final _input = TextEditingController();
  late Post? _post = widget.post;
  bool _sending = false;
  String _idemKey = Repo.newIdempotencyKey();

  @override
  void initState() {
    super.initState();
    if (_post == null) {
      ref.read(repoProvider).post(widget.postId).then((p) {
        if (mounted) setState(() => _post = p);
      }, onError: (Object _) {});
    }
  }

  @override
  void dispose() {
    _list.dispose();
    _input.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    final body = _input.text.trim();
    if (body.isEmpty || _sending) return;
    setState(() => _sending = true);
    try {
      final c = await ref.read(repoProvider).addComment(widget.postId, body, _idemKey);
      _idemKey = Repo.newIdempotencyKey();
      _input.clear();
      _list.insertFirst(c);
      if (_post != null) setState(() => _post = _post!.copyWith(commentCount: _post!.commentCount + 1));
    } on ApiException catch (e) {
      if (mounted) toast(context, e.title);
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  Future<void> _menu(Comment c) async {
    final choice = await showActionSheet<String>(context, [
      if (c.canDelete) (Sym.delete, 'حذف نظر', 'delete', true),
      if (c.author.id != ref.read(sessionProvider).userId) (Sym.flag, 'گزارش نظر', 'report', true),
      (Sym.person, 'مشاهده نمایه', 'profile', false),
    ]);
    if (!mounted || choice == null) return;
    switch (choice) {
      case 'delete':
        try {
          await ref.read(repoProvider).deleteComment(c.id);
          _list.remove((x) => x.id == c.id);
          if (_post != null) setState(() => _post = _post!.copyWith(commentCount: _post!.commentCount - 1));
          if (mounted) toast(context, 'نظر حذف شد.');
        } on ApiException catch (e) {
          if (mounted) toast(context, e.title);
        }
      case 'report':
        showReportSheet(context, targetType: 'comment', targetId: c.id, title: c.author.name, subtitle: c.body);
      case 'profile':
        context.push('/u/${c.author.id}');
    }
  }

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final post = _post;
    final count = post?.commentCount;
    return Scaffold(
      backgroundColor: cs.surface,
      resizeToAvoidBottomInset: true,
      appBar: ArchiveHeader(
        title: 'دیدگاه‌های پست',
        horizontal: 12,
        backSize: 40,
        shadow: const BoxShadow(color: Color(0x59000000), blurRadius: 12, offset: Offset(0, 1)),
        actions: [
          RoundIconButton(
            icon: Sym.shoppingCart,
            label: 'سبد خرید',
            size: 40,
            iconSize: 20,
            background: cs.surfaceContainerHigh,
            border: Border.all(color: cs.outlineVariant.withValues(alpha: 0.3)),
            onTap: () => soon(context, 'سبد خرید', 'فاز ۷'),
          ),
          const SizedBox(width: 6),
          RoundIconButton(icon: Sym.share, label: 'اشتراک‌گذاری', size: 40, iconSize: 22, color: cs.onSurfaceVariant, onTap: () => soon(context, 'اشتراک‌گذاری', 'فاز ۳')),
        ],
      ),
      body: Stack(children: [
        // dimmed post image behind the sheet
        Positioned(
          top: 0,
          left: 0,
          right: 0,
          height: 144,
          child: Opacity(
            opacity: 0.4,
            child: Stack(fit: StackFit.expand, children: [
              if (post != null && post.media.isNotEmpty) NetImage(post.media.first.url) else ColoredBox(color: cs.surfaceContainer),
              DecoratedBox(
                decoration: BoxDecoration(
                  gradient: LinearGradient(begin: Alignment.topCenter, end: Alignment.bottomCenter, colors: [cs.surface.withValues(alpha: 0.4), cs.surface.withValues(alpha: 0.85), cs.surfaceContainerLowest]),
                ),
              ),
            ]),
          ),
        ),
        Positioned.fill(
          top: 144,
          child: Container(
            decoration: BoxDecoration(
              color: cs.surfaceContainerLow,
              borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
              boxShadow: const [BoxShadow(color: Color(0x40000000), blurRadius: 50, offset: Offset(0, 25), spreadRadius: -12)],
            ),
            child: Column(children: [
              const SizedBox(height: 12),
              Container(width: 40, height: 6, decoration: BoxDecoration(color: cs.surfaceContainerHighest, borderRadius: BorderRadius.circular(999))),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                child: Row(children: [
                  Text('دیدگاه‌ها', style: tw(18, 24, 600, cs.onSurface)),
                  const SizedBox(width: 6),
                  if (count != null)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(color: cs.surfaceContainerHigh, borderRadius: BorderRadius.circular(999)),
                      child: Text(formatNumber(count), key: const Key('comment-count'), style: tw(12, 16, 600, cs.primary)),
                    ),
                  const Spacer(),
                  RoundIconButton(icon: Sym.sort, label: 'مرتب‌سازی', size: 36, iconSize: 20, color: cs.onSurfaceVariant, onTap: () => soon(context, 'مرتب‌سازی نظرات', 'نسخه‌های بعد')),
                  const SizedBox(width: 4),
                  RoundIconButton(icon: Sym.close, label: 'بستن', size: 36, iconSize: 20, color: cs.onSurfaceVariant, onTap: () => context.pop()),
                ]),
              ),
              Expanded(
                child: ListenableBuilder(
                  listenable: _list,
                  builder: (context, _) {
                    if (!_list.loadedOnce) return const LoadingView(rows: 4);
                    if (_list.error != null && _list.items.isEmpty) return Center(child: ErrorView(error: _list.error!, onRetry: _list.refresh));
                    if (_list.items.isEmpty) return const Center(child: EmptyView(icon: Sym.forum, title: 'هنوز نظری ثبت نشده', message: 'اولین نفری باشید که نظر می‌دهد.'));
                    return NotificationListener<ScrollNotification>(
                      onNotification: (n) {
                        if (n.metrics.extentAfter < 300) Future.microtask(_list.more);
                        return false;
                      },
                      child: ListView.separated(
                        padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                        itemCount: _list.items.length + (_list.done ? 0 : 1),
                        separatorBuilder: (_, _) => const SizedBox(height: 12),
                        itemBuilder: (_, i) {
                          if (i >= _list.items.length) {
                            return Center(
                              child: Container(
                                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                                decoration: BoxDecoration(color: cs.surfaceContainer, borderRadius: BorderRadius.circular(999)),
                                child: Row(mainAxisSize: MainAxisSize.min, children: [
                                  Sy(Sym.forum, size: 14, color: cs.primary),
                                  const SizedBox(width: 4),
                                  Text('در حال بارگذاری نظرات بیشتر...', style: tw(11, 14, 500, cs.onSurfaceVariant)),
                                ]),
                              ),
                            );
                          }
                          return _CommentCard(comment: _list.items[i], onMore: () => _menu(_list.items[i]));
                        },
                      ),
                    );
                  },
                ),
              ),
              if (post != null && !post.commentsAllowed)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text('امکان ثبت نظر برای این پست وجود ندارد.', style: tw(12, 18, 600, cs.onSurfaceVariant), textAlign: TextAlign.center),
                )
              else ...[
                SizedBox(
                  height: 54,
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    child: Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                      for (final e in const ['❤️', '🔥', '👏', '😍', '🛍️', '💯'])
                        GestureDetector(
                          onTap: () {
                            _input.text = '${_input.text}$e';
                            _input.selection = TextSelection.collapsed(offset: _input.text.length);
                            setState(() {});
                          },
                          child: Container(
                            width: 36,
                            height: 36,
                            decoration: BoxDecoration(color: cs.surfaceContainer, shape: BoxShape.circle),
                            alignment: Alignment.center,
                            child: Text(e, style: const TextStyle(fontSize: 18, height: 1.2)),
                          ),
                        ),
                    ]),
                  ),
                ),
                SafeArea(
                  top: false,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
                    child: Container(
                      height: 48,
                      decoration: BoxDecoration(color: cs.surfaceContainerHigh, borderRadius: BorderRadius.circular(12)),
                      padding: const EdgeInsets.symmetric(horizontal: 6),
                      child: Row(children: [
                        Avatar(author: Author(id: '', handle: ref.read(sessionProvider).handle ?? '', displayName: ref.read(sessionProvider).handle ?? ''), size: 36),
                        const SizedBox(width: 4),
                        Expanded(
                          child: TextField(
                            key: const Key('comment-input'),
                            controller: _input,
                            minLines: 1,
                            maxLines: 1,
                            textInputAction: TextInputAction.send,
                            onSubmitted: (_) => _send(),
                            inputFormatters: [LengthLimitingTextInputFormatter(1000)],
                            style: tw(14, 20, 400, cs.onSurface),
                            cursorColor: cs.primaryContainer,
                            decoration: InputDecoration(
                              isCollapsed: true,
                              border: InputBorder.none,
                              hintText: 'افزودن دیدگاه یا سوال درباره این پست...',
                              hintStyle: tw(14, 20, 400, cs.onSurfaceVariant.withValues(alpha: 0.7)),
                            ),
                          ),
                        ),
                        RoundIconButton(icon: Sym.alternateEmail, label: 'منشن', size: 32, iconSize: 20, color: cs.onSurfaceVariant, onTap: () => soon(context, 'منشن', 'فاز ۵')),
                        RoundIconButton(icon: Sym.mood, label: 'ایموجی', size: 32, iconSize: 20, color: cs.onSurfaceVariant, onTap: () => soon(context, 'صفحه‌کلید ایموجی', 'نسخه‌های بعد')),
                        const SizedBox(width: 4),
                        Material(
                          color: cs.primaryContainer,
                          borderRadius: BorderRadius.circular(8),
                          child: InkWell(
                            key: const Key('comment-send'),
                            borderRadius: BorderRadius.circular(8),
                            onTap: _sending ? null : _send,
                            child: const SizedBox(width: 36, height: 36, child: Center(child: Sy(Sym.send, size: 20, color: Colors.white, semantic: 'ارسال'))),
                          ),
                        ),
                      ]),
                    ),
                  ),
                ),
              ],
            ]),
          ),
        ),
      ]),
    );
  }
}

class _CommentCard extends StatelessWidget {
  const _CommentCard({required this.comment, required this.onMore});
  final Comment comment;
  final VoidCallback onMore;

  @override
  Widget build(BuildContext context) {
    final cs = context.cs;
    final c = comment;
    return Container(
      key: Key('comment-${c.id}'),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(color: cs.surfaceContainer.withValues(alpha: 0.6), borderRadius: BorderRadius.circular(12)),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        GestureDetector(onTap: () => context.push('/u/${c.author.id}'), child: Avatar(author: c.author, size: 36)),
        const SizedBox(width: 8),
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Row(children: [
              Flexible(child: Text(c.author.name, style: tw(12, 16, 600, cs.onSurface), overflow: TextOverflow.ellipsis)),
              if (c.isPostAuthor) ...[
                const SizedBox(width: 4),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  decoration: BoxDecoration(color: cs.surfaceContainerHigh, borderRadius: BorderRadius.circular(4)),
                  child: Text('نویسنده', style: tw(11, 14, 500, cs.primary)),
                ),
              ],
              const SizedBox(width: 8),
              Text(formatRelative(c.createdAt), style: tw(12, 18, 400, cs.onSurfaceVariant)),
              const Spacer(),
              GestureDetector(onTap: onMore, child: Semantics(button: true, label: 'گزینه‌های نظر', child: Sy(Sym.moreHoriz, size: 18, color: cs.onSurfaceVariant))),
            ]),
            const SizedBox(height: 4),
            Text(c.body, style: tw(14, 20, 400, cs.onSurface)),
            const SizedBox(height: 8),
            Row(children: [
              GestureDetector(onTap: () => soon(context, 'پسندیدن نظر', 'نسخه‌های بعد'), child: Sy(Sym.favorite, size: 16, color: cs.onSurfaceVariant)),
              const SizedBox(width: 16),
              GestureDetector(onTap: () => soon(context, 'پاسخ به نظر', 'نسخه‌های بعد'), child: Text('پاسخ', style: tw(11, 14, 500, cs.onSurfaceVariant))),
            ]),
          ]),
        ),
      ]),
    );
  }
}
