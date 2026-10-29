import 'package:flutter/foundation.dart';

import '../api/models.dart';

/// Keyset-paginated list state (doc 07 §1 cursors): first load, next pages, refresh, error and end-of-list.
class Paged<T> extends ChangeNotifier {
  Paged(this._fetch);
  final Future<ApiPage<T>> Function(String? cursor) _fetch;

  final List<T> items = [];
  String? _cursor;
  bool loading = false, done = false, loadedOnce = false;
  Object? error;
  bool _disposed = false;

  bool get isEmpty => loadedOnce && items.isEmpty && error == null;

  Future<void> refresh() async {
    _cursor = null;
    done = false;
    error = null;
    loading = false;
    await more(reset: true);
  }

  Future<void> more({bool reset = false}) async {
    if (loading || (done && !reset)) return;
    loading = true;
    error = null;
    _notify();
    try {
      final page = await _fetch(reset ? null : _cursor);
      if (reset) items.clear();
      items.addAll(page.items);
      _cursor = page.next;
      done = page.next == null;
    } catch (e) {
      error = e;
    } finally {
      loading = false;
      loadedOnce = true;
      _notify();
    }
  }

  void replace(bool Function(T) where, T Function(T) update) {
    for (var i = 0; i < items.length; i++) {
      if (where(items[i])) items[i] = update(items[i]);
    }
    _notify();
  }

  void remove(bool Function(T) where) {
    items.removeWhere(where);
    _notify();
  }

  void insertFirst(T item) {
    items.insert(0, item);
    _notify();
  }

  void _notify() {
    if (!_disposed) notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
