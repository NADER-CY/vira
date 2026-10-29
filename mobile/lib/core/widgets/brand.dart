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

/// The centre "create" action of the bottom bar, as drawn in the archive (gradient rounded square).
class CreateButton extends StatelessWidget {
  const CreateButton({super.key, required this.onPressed});
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => Semantics(
        button: true,
        label: 'ایجاد',
        child: InkWell(
          onTap: onPressed,
          borderRadius: BorderRadius.circular(ViraTokens.radiusLg),
          child: Ink(
            width: 52,
            height: 52,
            decoration: BoxDecoration(
              gradient: const LinearGradient(colors: ViraTokens.gradientBrand, begin: AlignmentDirectional.topStart, end: AlignmentDirectional.bottomEnd),
              borderRadius: BorderRadius.circular(ViraTokens.radiusLg),
            ),
            child: const Icon(Icons.add, color: Colors.white, size: 30),
          ),
        ),
      );
}
