import 'dart:typed_data';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';

/// A picked image: bytes + file name. The server re-encodes and strips metadata (EXIF/GPS) on upload.
typedef PickedImage = ({Uint8List bytes, String name});

/// Gallery picker; tests override it with fixed bytes.
abstract class PhotoPicker {
  Future<List<PickedImage>> pick({int max = 1});
}

class GalleryPicker implements PhotoPicker {
  const GalleryPicker();

  @override
  Future<List<PickedImage>> pick({int max = 1}) async {
    final picker = ImagePicker();
    final files = max == 1
        ? [?await picker.pickImage(source: ImageSource.gallery, maxWidth: 2048, maxHeight: 2048, imageQuality: 90)]
        : await picker.pickMultiImage(maxWidth: 2048, maxHeight: 2048, imageQuality: 90, limit: max);
    return [for (final f in files.take(max)) (bytes: await f.readAsBytes(), name: f.name)];
  }
}

final imagePickerProvider = Provider<PhotoPicker>((ref) => const GalleryPicker());
