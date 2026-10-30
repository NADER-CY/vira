import 'package:flutter/material.dart';

import '../theme/tokens.g.dart';

/// VIRA wordmark drawn with the brand gradient (no image assets needed).
class ViraWordmark extends StatelessWidget {
  const ViraWordmark({super.key, this.size = 26});
  final double size;

  @override
  Widget build(BuildContext context) => ShaderMask(
        shaderCallback: (r) => const LinearGradient(colors: ViraTokens.gradientBrand).createShader(r),
        child: Text('ویرا', style: TextStyle(fontFamily: ViraTokens.fontFamily, fontSize: size, fontWeight: FontWeight.w800, color: Colors.white)),
      );
}

/// The "V" mark of the archive's welcome screen (`otp`): inline SVG 48×48 — outer stroke 4.5 in currentColor,
/// inner stroke 3 and dot in #ffb2ba, with a soft glow.
class ViraMark extends StatelessWidget {
  const ViraMark({super.key, this.size = 48, required this.color, this.inner = const Color(0xFFFFB2BA)});
  final double size;
  final Color color, inner;

  @override
  Widget build(BuildContext context) => Semantics(
        label: 'نشان ویرا',
        image: true,
        child: CustomPaint(size: Size.square(size), painter: _MarkPainter(color, inner)),
      );
}

class _MarkPainter extends CustomPainter {
  _MarkPainter(this.color, this.inner);
  final Color color, inner;

  @override
  void paint(Canvas canvas, Size size) {
    final k = size.width / 48;
    Paint stroke(Color c, double w) => Paint()
      ..color = c
      ..style = PaintingStyle.stroke
      ..strokeWidth = w * k
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    final outer = Path()
      ..moveTo(10 * k, 12 * k)
      ..lineTo(24 * k, 38 * k)
      ..lineTo(38 * k, 12 * k);
    final inn = Path()
      ..moveTo(19 * k, 12 * k)
      ..lineTo(24 * k, 23 * k)
      ..lineTo(29 * k, 12 * k);
    // drop-shadow-[0_0_12px_rgba(255,79,116,0.6)]
    canvas.drawPath(outer, stroke(const Color(0x99FF4F74), 4.5)..maskFilter = const MaskFilter.blur(BlurStyle.normal, 6));
    canvas.drawPath(outer, stroke(color, 4.5));
    canvas.drawPath(inn, stroke(inner, 3));
    canvas.drawCircle(Offset(24 * k, 38 * k), 2.5 * k, Paint()..color = inner);
  }

  @override
  bool shouldRepaint(_MarkPainter old) => old.color != color || old.inner != inner;
}
