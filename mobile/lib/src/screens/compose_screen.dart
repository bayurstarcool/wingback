import 'package:flutter/material.dart';

import '../api/messages_api.dart';

/// Basic compose screen: recipient, message body, carrier choice, and a
/// submit button that calls the backend. GPS geolocation and live-map
/// ETA rendering are intentionally left as TODOs — this scaffold gives
/// the compose contract a working baseline to build on.
class ComposeScreen extends StatefulWidget {
  final MessagesApi api;
  const ComposeScreen({super.key, required this.api});

  @override
  State<ComposeScreen> createState() => _ComposeScreenState();
}

class _ComposeScreenState extends State<ComposeScreen> {
  final _recipientController = TextEditingController();
  final _bodyController = TextEditingController();
  String _carrierSlug = 'pigeon';
  bool _sending = false;
  String? _error;
  ComposeResult? _result;

  static const _carriers = [
    ('pigeon', 'Merpati Pos', '🕊️'),
    ('paper_plane', 'Pesawat Kertas', '✈️'),
    ('falcon', 'Elang Perkasa', '🦅'),
    ('drone', 'Mini Drone', '🚁'),
  ];

  @override
  void dispose() {
    _recipientController.dispose();
    _bodyController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    setState(() {
      _error = null;
      _result = null;
    });

    if (_recipientController.text.trim().isEmpty ||
        _bodyController.text.trim().isEmpty) {
      setState(() => _error = 'Isi penerima dan pesan dulu.');
      return;
    }

    setState(() => _sending = true);
    try {
      // TODO: replace hardcoded coordinates with geolocator package output.
      final result = await widget.api.compose(
        recipientId: _recipientController.text.trim(),
        body: _bodyController.text.trim(),
        carrierSlug: _carrierSlug,
        senderLat: -6.2088,
        senderLng: 106.8456,
        recipientLat: -7.2575,
        recipientLng: 112.7521,
      );
      setState(() => _result = result);
    } catch (e) {
      setState(() => _error = 'Gagal mengirim: $e');
    } finally {
      setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Wingback')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: ListView(
          children: [
            TextField(
              controller: _recipientController,
              decoration: const InputDecoration(labelText: 'ID Penerima'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _bodyController,
              maxLength: 2000,
              maxLines: 4,
              decoration: const InputDecoration(labelText: 'Pesan'),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: _carriers.map((c) {
                final (slug, name, emoji) = c;
                return ChoiceChip(
                  label: Text('$emoji $name'),
                  selected: _carrierSlug == slug,
                  onSelected: (_) => setState(() => _carrierSlug = slug),
                );
              }).toList(),
            ),
            const SizedBox(height: 20),
            if (_error != null)
              Text(_error!, style: const TextStyle(color: Colors.red)),
            FilledButton(
              onPressed: _sending ? null : _submit,
              child: Text(_sending ? 'Mengirim...' : 'Kirim'),
            ),
            if (_result != null) ...[
              const SizedBox(height: 20),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Jarak: ${_result!.distanceKm.toStringAsFixed(1)} km'),
                      Text('Kecepatan: ${_result!.speedKmh} km/j'),
                      Text('Tiba: ${_result!.arrivesAt}'),
                      if (_result!.willBeLost)
                        const Text(
                          '⚠️ Ada kemungkinan hilang di tengah jalan',
                          style: TextStyle(color: Colors.orange),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
