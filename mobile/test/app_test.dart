import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'helpers.dart';

Map<String, dynamic> bodyOf(FakeApi api, String path, [String method = 'POST']) =>
    jsonDecode(api.requests.lastWhere((r) => r.url.path == path && r.method == method).body) as Map<String, dynamic>;

bool called(FakeApi api, String path, [String method = 'GET']) => api.requests.any((r) => r.url.path == path && r.method == method);

Future<void> scrollTo(WidgetTester tester, Finder f) async {
  await tester.scrollUntilVisible(f, 200, scrollable: find.byType(Scrollable).first);
  await tester.pumpAndSettle();
}

void main() {
  group('identity', () {
    testWidgets('signed out → welcome; RTL; consents start unticked and gate the button (UC-5)', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, signedIn: false);
      expect(find.text('احراز هویت پیامکی'), findsOneWidget);
      final ctx = tester.element(find.text('احراز هویت پیامکی'));
      expect(Directionality.of(ctx), TextDirection.rtl);
      for (final k in ['consent-terms', 'consent-privacy', 'consent-age']) {
        expect(tester.widget<Checkbox>(find.descendant(of: find.byKey(Key(k)), matching: find.byType(Checkbox))).value, isFalse);
      }
      await tester.enterText(find.byKey(const Key('phone')), '۰۹۱۲۳۴۵۶۷۸۹');
      await tester.pump();
      await tester.tap(find.byKey(const Key('welcome-submit')));
      await tester.pumpAndSettle();
      expect(find.text('کد تأیید را وارد کنید'), findsNothing, reason: 'button disabled until the boxes are ticked');
    });

    testWidgets('sign-up: phone → 6-digit code → home; consents and device are sent', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, signedIn: false);
      await tester.enterText(find.byKey(const Key('phone')), '09123456789');
      for (final k in ['consent-terms', 'consent-privacy', 'consent-age']) {
        await scrollTo(tester, find.byKey(Key(k)));
        await tester.tap(find.byKey(Key(k)));
      }
      await tester.pump();
      await scrollTo(tester, find.byKey(const Key('welcome-submit')));
      await tester.tap(find.byKey(const Key('welcome-submit')));
      await tester.pumpAndSettle();
      expect(bodyOf(api, '/v1/auth/otp')['phone'], '09123456789');
      expect(find.text('کد تأیید را وارد کنید'), findsOneWidget);
      expect(find.textContaining('کد ۶ رقمی'), findsOneWidget);
      await tester.enterText(find.byKey(const Key('otp-input')), '123456');
      await tester.pumpAndSettle();
      final v = bodyOf(api, '/v1/auth/otp/verify');
      expect(v['code'], '123456');
      expect(v['consents'], {'terms': true, 'privacy': true, 'age_confirmation': true, 'text_version': 'p2-draft-1'});
      expect(v['intends_to_sell'], isFalse);
      expect((v['device'] as Map)['id'], isNotEmpty);
      expect(find.text('برای شما'), findsOneWidget, reason: 'landed on the home feed');
    });

    testWidgets('wrong code and lockout messages', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, signedIn: false);
      await tester.enterText(find.byKey(const Key('phone')), '09123456789');
      for (final k in ['consent-terms', 'consent-privacy', 'consent-age']) {
        await scrollTo(tester, find.byKey(Key(k)));
        await tester.tap(find.byKey(Key(k)));
      }
      await scrollTo(tester, find.byKey(const Key('welcome-submit')));
      await tester.tap(find.byKey(const Key('welcome-submit')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byKey(const Key('otp-input')), '111111');
      await tester.pumpAndSettle();
      expect(find.text('کد واردشده درست نیست.'), findsOneWidget);
      await tester.enterText(find.byKey(const Key('otp-input')), '999999');
      await tester.pumpAndSettle();
      expect(find.textContaining('قفل شد'), findsOneWidget);
    });

    testWidgets('after a restart the stored refresh token is rotated before the first call', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester);
      expect(api.refreshCalls, 1);
      final feed = api.requests.firstWhere((r) => r.url.path == '/v1/feed/home');
      expect(feed.headers['Authorization'], 'Bearer at2');
      expect(feed.headers['X-Device-Id'], isNotEmpty);
    });

    testWidgets('logout revokes the session and returns to welcome', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/settings/account');
      await tester.tap(find.byKey(const Key('logout-button')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/auth/logout', 'POST'), isTrue);
      expect(find.text('احراز هویت پیامکی'), findsOneWidget);
    });

    testWidgets('account deletion needs the acknowledgement box', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/settings/account');
      await scrollTo(tester, find.byKey(const Key('delete-button')));
      await tester.tap(find.byKey(const Key('delete-button')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/me/deletion', 'POST'), isFalse);
      await scrollTo(tester, find.byKey(const Key('delete-ack')));
      await tester.tap(find.byKey(const Key('delete-ack')));
      await tester.pump();
      await scrollTo(tester, find.byKey(const Key('delete-button')));
      await tester.tap(find.byKey(const Key('delete-button')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('ارسال درخواست حذف'));
      await tester.pumpAndSettle();
      expect(bodyOf(api, '/v1/me/deletion'), {'confirm': true});
      expect(find.text('احراز هویت پیامکی'), findsOneWidget);
    });

    testWidgets('sessions: other device can be ended', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/settings/security');
      expect(find.text('این دستگاه (آنلاین)'), findsOneWidget);
      await scrollTo(tester, find.byKey(const Key('end-s2')));
      await tester.tap(find.byKey(const Key('end-s2')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('خاتمه نشست').last);
      await tester.pumpAndSettle();
      expect(called(api, '/v1/me/sessions/s2', 'DELETE'), isTrue);
    });
  });

  group('feed', () {
    testWidgets('home shows posts; like is sent and reflected', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester);
      expect(find.textContaining('ترکیب رنگ لیمویی'), findsOneWidget);
      expect(find.text('#استایل_تابستانه'), findsOneWidget);
      expect(find.text('۱ / ۳'), findsOneWidget);
      // The action row sits under the bottom bar on the test phone: scroll the feed (vertical scrollable) up.
      await tester.drag(find.byWidgetPredicate((w) => w is Scrollable && w.axisDirection == AxisDirection.down).first, const Offset(0, -260));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('like-p0')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/posts/p0/like', 'PUT'), isTrue);
      expect(find.text('۳٬۴۲۱'), findsOneWidget);
    });

    testWidgets('following tab loads the following feed', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester);
      await tester.tap(find.byKey(const Key('tab-following')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/feed/following'), isTrue);
    });

    testWidgets('empty and error states', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, api: FakeApi(posts: 0));
      expect(find.text('فید شما خالی است'), findsOneWidget);
    });

    testWidgets('feed error shows retry', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, api: FakeApi(feedError: true));
      expect(find.text('سرویس موقتاً در دسترس نیست.'), findsOneWidget);
      expect(find.text('تلاش دوباره'), findsOneWidget);
    });

    testWidgets('server unreachable → error state', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, serverUp: false);
      expect(find.textContaining('اتصال به سرور برقرار نشد'), findsWidgets);
    });

    testWidgets('comments: list, add with idempotency key', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/post/p0/comments');
      expect(find.textContaining('کیفیت پارچه‌اش عالیه'), findsOneWidget);
      await tester.enterText(find.byKey(const Key('comment-input')), 'خیلی قشنگه');
      await tester.tap(find.byKey(const Key('comment-send')));
      await tester.pumpAndSettle();
      final req = api.requests.lastWhere((r) => r.url.path == '/v1/posts/p0/comments' && r.method == 'POST');
      expect(req.headers['Idempotency-Key'], isNotEmpty);
      expect(find.text('خیلی قشنگه'), findsOneWidget);
    });

    testWidgets('create post: pick image → upload → publish', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/post/new');
      await tester.tap(find.byKey(const Key('add-slide')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/media/images', 'POST'), isTrue);
      await tester.enterText(find.byKey(const Key('caption')), 'اولین پست #ویرا');
      await scrollTo(tester, find.byKey(const Key('publish')));
      await tester.tap(find.byKey(const Key('publish')));
      await tester.pumpAndSettle();
      final b = bodyOf(api, '/v1/posts');
      expect(b['media_ids'], ['up1']);
      expect(b['visibility'], 'public');
      expect(api.requests.lastWhere((r) => r.url.path == '/v1/posts').headers['Idempotency-Key'], isNotEmpty);
    });

    testWidgets('report a post from its menu', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/post/p0');
      await tester.tap(find.bySemanticsLabel('گزینه‌ها').first);
      await tester.pumpAndSettle();
      await tester.tap(find.text('گزارش پست'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('reason-spam')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('report-send')));
      await tester.pumpAndSettle();
      expect(api.reports.single, {'target_type': 'post', 'target_id': 'p0', 'reason': 'spam'});
    });
  });

  group('profile & graph', () {
    testWidgets('own profile; edit sends If-Match and shows field errors', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/profile');
      expect(find.text('۳۸٬۵۰۰'), findsOneWidget);
      await tester.tap(find.byKey(const Key('edit-profile')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('masked-phone')), findsOneWidget);
      await tester.enterText(find.byKey(const Key('field-handle')), 'taken');
      await scrollTo(tester, find.byKey(const Key('save-profile')));
      await tester.tap(find.byKey(const Key('save-profile')));
      await tester.pumpAndSettle();
      expect(api.requests.lastWhere((r) => r.method == 'PATCH').headers['If-Match'], '"3"');
      await scrollTo(tester, find.text('این نام کاربری قبلاً انتخاب شده است.'));
      expect(find.text('این نام کاربری قبلاً انتخاب شده است.'), findsOneWidget);
    });

    testWidgets('follow another account', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/u/$other');
      await tester.tap(find.byKey(const Key('follow-button')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/users/$other/follow', 'POST'), isTrue);
      expect(find.text('دنبال می‌کنید'), findsOneWidget);
    });

    testWidgets('private account hides posts until followed', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, api: FakeApi(privateOther: true), location: '/u/$other');
      expect(find.text('این حساب خصوصی است'), findsOneWidget);
    });

    testWidgets('follow requests can be approved', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/activity');
      expect(find.textContaining('۲ نفر منتظر'), findsOneWidget);
      await tester.tap(find.byKey(const Key('approve-u-req1')));
      await tester.pumpAndSettle();
      expect(called(api, '/v1/me/follow-requests/u-req1/approve', 'POST'), isTrue);
      expect(find.byKey(const Key('request-u-req1')), findsNothing);
    });

    testWidgets('privacy: private account switch is saved', (tester) async {
      await setPhone(tester);
      final api = await pumpApp(tester, location: '/settings/privacy');
      await tester.tap(find.byKey(const Key('private-switch')));
      await tester.pumpAndSettle();
      expect(bodyOf(api, '/v1/me/privacy', 'PUT')['is_private'], isTrue);
      await scrollTo(tester, find.byKey(const Key('comments-followers')));
      await tester.tap(find.byKey(const Key('comments-followers')));
      await tester.pumpAndSettle();
      expect(bodyOf(api, '/v1/me/privacy', 'PUT')['comments_from'], 'followers');
    });
  });

  group('shell & locked features', () {
    testWidgets('tabs; reels stays a Phase 3 placeholder; create sheet keeps later entries disabled', (tester) async {
      await setPhone(tester);
      await pumpApp(tester);
      expect(tester.getCenter(find.text('خانه').last).dx, greaterThan(tester.getCenter(find.text('پروفایل').last).dx));
      await tester.tap(find.text('ریلز').last);
      await tester.pumpAndSettle();
      expect(find.text('ریلز و ویدیوهای کوتاه'), findsOneWidget);
      await tester.tap(find.bySemanticsLabel('ایجاد'));
      await tester.pumpAndSettle();
      expect(find.text('پخش زنده'), findsOneWidget);
      expect(find.text('فاز ۸'), findsOneWidget);
    });

    testWidgets('theme switch light ↔ dark uses the archive palettes and is persisted', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, prefs: {'vira.theme': 'light'}, location: '/settings');
      BuildContext ctx() => tester.element(find.text('تنظیمات ظاهر برنامه'));
      expect(Theme.of(ctx()).colorScheme.surface, const Color(0xFFFAF9FF));
      await tester.tap(find.text('تیره'));
      await tester.pumpAndSettle();
      expect(Theme.of(ctx()).brightness, Brightness.dark);
      expect(Theme.of(ctx()).colorScheme.surface, const Color(0xFF0F131C));
      expect(Theme.of(ctx()).colorScheme.primary, const Color(0xFFFFB2BA));
    });

    testWidgets('wallet stays locked behind gate G1', (tester) async {
      await setPhone(tester);
      await pumpApp(tester, location: '/settings');
      await scrollTo(tester, find.text('کیف پول و کارت‌ها'));
      await tester.tap(find.text('کیف پول و کارت‌ها'));
      await tester.pump();
      // The tile's lock label and the snackbar both name the gate.
      expect(find.textContaining('کیف پول تا عبور از دروازهٔ G1'), findsOneWidget);
      expect(find.textContaining('دروازهٔ G1'), findsNWidgets(2));
    });
  });
}
