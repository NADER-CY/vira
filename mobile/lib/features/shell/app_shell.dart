import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/theme/symbols.g.dart';
import '../../core/theme/vira_theme.dart';
import '../../core/ui/kit.dart';
import '../create/create_sheet.dart';

/// Bottom bar as drawn in `_33` / `home_feed_light`: خانه · کاوش · ＋ · ریلز · پروفایل (h-16, justify-around).
/// In RTL the first destination (home) is on the right.
class AppShell extends StatelessWidget {
  const AppShell({super.key, required this.shell});
  final StatefulNavigationShell shell;

  @override
  Widget build(BuildContext context) {
    final sk = context.skin;
    final dark = context.isDark;
    Widget item(int branch, IconData icon, String label) {
      final active = shell.currentIndex == branch;
      final color = active ? sk.navActive : sk.navInactive;
      return Semantics(
        button: true,
        selected: active,
        label: label,
        child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: () => shell.goBranch(branch, initialLocation: active),
          child: SizedBox(
            width: 48,
            height: 48,
            child: Transform.scale(
              scale: active ? 1.05 : 1,
              child: ExcludeSemantics(
                child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                  Sy(icon, size: 24, color: color),
                  const SizedBox(height: 2),
                  Text(label, style: tw(11, dark ? 14 : 16.5, active && !dark ? 700 : (dark ? 500 : 400), color), maxLines: 1, softWrap: false),
                ]),
              ),
            ),
          ),
        ),
      );
    }

    return Scaffold(
      body: shell,
      bottomNavigationBar: Container(
        decoration: BoxDecoration(
          color: sk.navBg,
          border: sk.navBorder == null ? null : Border(top: BorderSide(color: sk.navBorder!)),
          boxShadow: sk.navShadow,
        ),
        child: SafeArea(
          top: false,
          child: SizedBox(
            height: 64,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                item(0, Sym.home, 'خانه'),
                item(1, Sym.explore, 'کاوش'),
                Semantics(
                  button: true,
                  label: 'ایجاد',
                  child: GestureDetector(
                    onTap: () => showCreateSheet(context),
                    child: SizedBox(
                      width: 48,
                      height: 48,
                      child: Center(
                        child: Container(
                          width: 40,
                          height: 40,
                          decoration: BoxDecoration(
                            color: sk.createBg,
                            gradient: sk.createGradient == null ? null : LinearGradient(colors: sk.createGradient!, begin: Alignment.topLeft, end: Alignment.bottomRight),
                            borderRadius: BorderRadius.circular(12),
                            boxShadow: const [BoxShadow(color: Color(0x0D000000), blurRadius: 2, offset: Offset(0, 1))],
                          ),
                          alignment: Alignment.center,
                          child: ExcludeSemantics(child: Sy(Sym.add, size: 26, color: sk.createFg)),
                        ),
                      ),
                    ),
                  ),
                ),
                item(2, Sym.movie, 'ریلز'),
                item(3, Sym.accountCircle, 'پروفایل'),
              ]),
            ),
          ),
        ),
      ),
    );
  }
}
