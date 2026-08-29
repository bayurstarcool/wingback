// Wingback mobile entry point.
//
// This file is hand-written scaffolding. After running `flutter create .`
// on a machine with the Flutter SDK (Android/iOS toolchains required),
// merge the generated android/ and ios/ folders with this lib/ tree.
import 'package:flutter/material.dart';

import 'src/api/messages_api.dart';
import 'src/screens/compose_screen.dart';

void main() {
  runApp(const WingbackApp());
}

class WingbackApp extends StatelessWidget {
  const WingbackApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Wingback',
      theme: ThemeData(
        colorSchemeSeed: const Color(0xFF6366F1),
        useMaterial3: true,
      ),
      home: ComposeScreen(api: MessagesApi()),
    );
  }
}
