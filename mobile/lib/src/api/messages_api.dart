import 'dart:convert';

import 'package:http/http.dart' as http;

/// Result of composing a message: mirrors the backend's composeResponse
/// JSON shape (see backend/internal/handlers/messages.go).
class ComposeResult {
  final String messageId;
  final double distanceKm;
  final double speedKmh;
  final DateTime departsAt;
  final DateTime arrivesAt;
  final bool willBeLost;

  ComposeResult({
    required this.messageId,
    required this.distanceKm,
    required this.speedKmh,
    required this.departsAt,
    required this.arrivesAt,
    required this.willBeLost,
  });

  factory ComposeResult.fromJson(Map<String, dynamic> json) => ComposeResult(
        messageId: json['message_id'] as String,
        distanceKm: (json['distance_km'] as num).toDouble(),
        speedKmh: (json['speed_kmh'] as num).toDouble(),
        departsAt: DateTime.parse(json['departs_at'] as String),
        arrivesAt: DateTime.parse(json['arrives_at'] as String),
        willBeLost: json['will_be_lost'] as bool,
      );
}

class ApiException implements Exception {
  final int statusCode;
  final String message;
  ApiException(this.statusCode, this.message);

  @override
  String toString() => 'ApiException($statusCode): $message';
}

/// Thin HTTP client for the Wingback backend. Base URL should be injected
/// via --dart-define=API_BASE_URL=... at build time; defaults to the
/// local dev server for emulator testing.
class MessagesApi {
  final String baseUrl;
  final http.Client _client;

  MessagesApi({
    String? baseUrl,
    http.Client? client,
  })  : baseUrl = baseUrl ??
            const String.fromEnvironment(
              'API_BASE_URL',
              defaultValue: 'http://10.0.2.2:8090',
            ),
        _client = client ?? http.Client();

  Future<ComposeResult> compose({
    required String recipientId,
    required String body,
    String? carrierSlug,
    required double senderLat,
    required double senderLng,
    required double recipientLat,
    required double recipientLng,
  }) async {
    final uri = Uri.parse('$baseUrl/api/messages');
    final response = await _client.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'recipient_id': recipientId,
        'body': body,
        if (carrierSlug != null) 'carrier_slug': carrierSlug,
        'sender_lat': senderLat,
        'sender_lng': senderLng,
        'recipient_lat': recipientLat,
        'recipient_lng': recipientLng,
      }),
    );

    if (response.statusCode != 201) {
      throw ApiException(response.statusCode, response.body);
    }

    return ComposeResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }
}
