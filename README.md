# Wingback

Pesan yang dibawa carrier virtual (merpati, elang, drone, dst) lewat jarak GPS **asli** antar pengguna — bukan simulasi fixed 177km/h. Terinspirasi dari tren Carrier Pidge/Roost (Gen Z slow-messaging), tapi ditambah monetisasi ads+skin dan mechanic collection yang mereka tidak punya.

## Struktur

```
backend/   Go (Echo) — delivery engine, REST API, WebSocket (rencana)
web/       SvelteKit — compose UI, live map, skin shop
mobile/    Flutter — scaffold manual, jalankan `flutter create .` di lokal
docs/      Catatan produk & arsitektur
```

## Core mechanic

Delivery time = haversine distance(sender, recipient) / carrier speed, dengan:
- Minimum floor 5 detik (biar tetap ada momen "terbang" walau sekota)
- 0,2% probabilitas pesan hilang (dikonfigurasi di `.env`)
- Rewarded-ad speedup: potong sisa waktu tunggu N% (default 50%, max 3x/hari)

Lihat `backend/internal/delivery/engine.go` untuk implementasi dan test.

## Menjalankan backend

```bash
cd backend
cp .env.example .env
go mod download
go run ./cmd/server
# server listen di :8080 (atau $PORT)
```

Test: `go test ./...`

Butuh Postgres + Redis untuk fitur penuh (lihat `migrations/001_init.sql`); scaffold saat ini fokus ke kontrak HTTP delivery engine dulu — persistence & WebSocket live-tracking belum diimplementasikan (lihat TODO di `internal/handlers/messages.go`).

## Menjalankan web

```bash
cd web
cp .env.example .env
npm install
npm run dev -- --open
```

Test: `npm run test` | Build: `npm run build` (pakai `@sveltejs/adapter-node`, jalankan hasil build dengan `node build`)

## Menjalankan mobile

Folder `mobile/` berisi scaffold `lib/` manual (belum ada `android/`/`ios/` karena Flutter SDK tidak tersedia di server ini). Di device dev:

```bash
cd mobile
flutter create .          # generate platform folders, merge dengan pubspec.yaml yang sudah ada
flutter pub get
flutter run --dart-define=API_BASE_URL=http://<ip-backend>:8080
```

## Status implementasi

- [x] Delivery engine (haversine, ETA, loss probability, ad-speedup) — full test coverage
- [x] REST endpoint `POST /api/messages` — compose + compute plan
- [x] Web compose UI (SvelteKit + Tailwind) — hit API, render countdown
- [x] Mobile compose UI scaffold (Flutter, belum di-`flutter create`)
- [ ] Persistence (Postgres) — schema ada, belum diwire ke handler
- [ ] WebSocket live-map tracking
- [ ] Auth (JWT scaffolded di config, belum ada endpoint)
- [ ] Skin shop / inventory / currency
- [ ] AdMob rewarded ad integration
- [ ] Push notification (FCM)

## Keamanan

Endpoint `/api/messages` saat ini **tanpa autentikasi** — siapa saja bisa compose pesan atas nama `recipient_id`/`sender` apa pun. Ini scaffold awal; wajib tambah auth (JWT sudah disiapkan di `internal/config`) sebelum deploy publik.
