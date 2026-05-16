# GoWA Dashboard

Bundle dua aplikasi Go untuk WhatsApp Web multi-device:

1. **`src/`** — **GoWA Core**. REST/MCP server WhatsApp Web berbasis [`whatsmeow`](https://go.mau.fi/whatsmeow). Ini adalah upstream [`aldinokemal/go-whatsapp-web-multidevice`](https://github.com/aldinokemal/go-whatsapp-web-multidevice) yang **tidak dimodifikasi** sehingga update versi bisa di-pull tanpa konflik.
2. **`dashboard/`** — **WhatsApp Dashboard**. Companion app berdiri sendiri (binary Go terpisah) yang menambahkan UI manajemen device + **penjadwalan pesan** (one-time, harian, mingguan, bulanan, tahunan, cron) + log eksekusi. Berkomunikasi dengan core via REST.

```
┌─────────────────────────┐      HTTP      ┌────────────────────────┐      HTTP      ┌────────────────────────┐
│  Browser (Vue 3 SPA)    │ ─────────────► │  dashboard (8088)      │ ─────────────► │  gowa-core (3000)      │
│  Semantic UI + axios    │                │  - SQLite jadwal & log │                │  - whatsmeow / WA Web  │
└─────────────────────────┘                │  - cron + one-shot     │                │  - REST + MCP          │
                                           └────────────────────────┘                └────────────────────────┘
```

---

## Daftar Isi

- [Fitur](#fitur)
- [Struktur Repo](#struktur-repo)
- [Cara Menjalankan](#cara-menjalankan)
- [Konfigurasi](#konfigurasi)
- [REST API Dashboard](#rest-api-dashboard)
- [Tipe Jadwal](#tipe-jadwal)
- [Catatan Penting](#catatan-penting)
- [Lisensi & Atribusi](#lisensi--atribusi)

---

## Fitur

### Core (`src/`)

Lihat [`how-to-use.md`](./how-to-use.md) untuk dokumentasi lengkap upstream. Ringkasan fitur utama:

- **Multi-device** dalam satu instance — pairing via QR atau kode telepon, scope tiap request pakai header `X-Device-Id` atau query `device_id`.
- **Kirim pesan** semua tipe: text, image, video, audio, file/document, location, link preview, contact, poll, sticker (auto-convert WebP), reaction, edit, revoke, forward.
- **Mention** biasa (`@628xxx`), **ghost mention** (tidak tampil `@` di teks), dan keyword `@everyone` untuk seluruh anggota grup.
- **Group management**: buat grup, invite/kick/promote/demote, ubah subject/description, invite-link, leave.
- **WhatsApp Status / Story** posting.
- **Auto reply**, **auto mark-read**, **auto download media**, **auto reject call** (semua opsional via flag/env).
- **AI Auto-Reply dengan RAG** (per-device, opt-in per chat) — multi-provider: Anthropic (Claude) atau OpenAI-compatible (OpenAI/OpenRouter/Sumopod/DeepSeek/Groq/Ollama). Upload PDF/DOCX/TXT/MD sebagai knowledgebase, di-chunk + di-embed otomatis (sqlite-vec). Style preset (formal/casual/technical/custom), guardrail anti-halusinasi, rate limit per-chat, typing indicator selama LLM generate, audit log lengkap. API key tersimpan terenkripsi AES-GCM via `AI_ENCRYPTION_KEY`.
- **Webhook** outbound dengan HMAC signature dan **event filtering** (`message`, `message.ack`, `message.reaction`, `group.participants`, `call.offer`, dll.).
- **Chatwoot** bidirectional sync (incoming → inbox, outgoing → conversation).
- **MCP server mode** (`./whatsapp mcp`) untuk integrasi dengan AI agent via Model Context Protocol.
- **Storage** SQLite (default) atau PostgreSQL via `DB_URI`.
- **Subpath deployment** (`--base-path=/gowa`), **basic auth multi-user**, **trusted proxies**, **debug mode**.

> REST mode dan MCP mode tidak bisa jalan bersamaan dalam satu proses (keterbatasan whatsmeow).

### Dashboard (`dashboard/`)

UI web (Vue 3 + Semantic UI, di-embed via `go:embed`) dengan lima tab:

| Tab                  | Fungsi                                                                                                          |
|----------------------|-----------------------------------------------------------------------------------------------------------------|
| **Devices**          | List device upstream, tambah device baru, login QR (di-proxy melalui dashboard — core tidak perlu publik), login via kode telepon, logout, reconnect, hapus device. |
| **Kirim Sekarang**   | Form kirim instan: text, image, video, file, audio, location, link. Pilih device & tujuan (nomor / JID grup).   |
| **Jadwal & Reminder**| CRUD jadwal: enable/disable, preview 5 fire-time berikutnya, tombol "Run Now" untuk uji manual, kolom next-run. |
| **Riwayat**          | Log eksekusi global + per-jadwal (status sukses/error, response upstream, pesan error).                          |
| **AI Reply**         | 4 sub-section yang nge-proxy ke core: **Config** (provider/model/prompt style/API key dengan masked-key indicator + test connection), **Knowledgebase** (upload PDF/DOCX/TXT/MD + list + reindex + delete), **Chat Toggle** (opt-in per chat JID, auto-format nomor `08xxx`/`62xxx` → `@s.whatsapp.net`), **Logs** (audit eksekusi dengan filter chat/status). Setting tersimpan di core (per-device, encrypted at rest). |

Kemampuan inti dashboard:

- **Penjadwalan fleksibel** — `once` / `daily` / `weekly` (multi-pilihan hari) / `monthly` / `yearly` / `cron` (5-field [`robfig/cron/v3`](https://github.com/robfig/cron)).
- **Timezone-aware** — tiap jadwal punya zona waktu sendiri (default dari `DASHBOARD_TZ`, mis. `Asia/Jakarta`).
- **Persisten** — SQLite pure-Go (`modernc.org/sqlite`), tidak butuh CGO, binary statis kecil.
- **One-shot survive restart** — schedule type `once` yang terlewat saat downtime tetap di-fire setelah dashboard start ulang (status "missed run").
- **QR proxy** — endpoint `/api/qr/:filename` mem-fetch PNG QR dari core, sehingga browser tidak perlu akses langsung ke port core (cocok di belakang reverse proxy / private network).
- **Basic auth opsional** terpisah dari upstream (`DASHBOARD_BASIC_AUTH=user:pass`).
- **Health probe** `/api/_health` untuk verifikasi build/version + cek konektivitas upstream.
- **Tidak menyentuh `src/`** — semua state dashboard di `dashboard/data/dashboard.db`, upgrade core cukup `git pull` upstream.

---

## Struktur Repo

```
.
├── src/                        # GoWA Core (upstream, jangan dimodifikasi)
│   ├── cmd/                    # Cobra CLI: rest, mcp subcommand
│   ├── domains/                # Interface + DTO (contract-only)
│   ├── infrastructure/         # whatsmeow, chatstorage, chatwoot
│   ├── usecase/                # Business logic
│   ├── ui/{rest,mcp}/          # Fiber HTTP & MCP handler
│   ├── views/                  # Vue.js 3 (Semantic UI) — UI core
│   └── ...
├── dashboard/                  # Companion app — binary terpisah
│   ├── main.go                 # Fiber app, embed web/
│   ├── internal/
│   │   ├── api/                # REST handler /api/*
│   │   ├── config/             # Env loader (godotenv)
│   │   ├── scheduler/          # robfig/cron + time.Timer
│   │   ├── store/              # SQLite (modernc.org/sqlite)
│   │   └── wa/                 # Client HTTP ke core
│   ├── web/index.html          # SPA Vue 3 (di-embed)
│   ├── Dockerfile              # Multi-stage alpine, non-root uid 20001
│   └── entrypoint.sh
├── docker/golang.Dockerfile    # Image core
├── docker-compose.yml          # Hanya core
├── docker-compose.full.yml     # Core + dashboard
├── docker-compose.aapanel.yml  # Versi untuk aaPanel (bind 127.0.0.1 + reverse proxy)
├── docs/                       # OpenAPI core, dokumentasi webhook & Chatwoot
├── how-to-use.md               # Manual lengkap core
└── readme.md                   # File ini
```

---

## Cara Menjalankan

### Opsi A — Docker Compose (rekomendasi)

```bash
# Build & jalankan core + dashboard sekaligus
docker compose -f docker-compose.full.yml up -d --build
```

- Core   → http://localhost:3000 (UI bawaan untuk pairing & operasi langsung)
- Dashboard → http://localhost:8088

Untuk deploy di aaPanel (port di-bind ke loopback supaya tidak bentrok, expose via Nginx reverse proxy):

```bash
docker compose -f docker-compose.aapanel.yml up -d --build
```

### Opsi B — Lokal tanpa Docker

Butuh **Go 1.25+** dan **FFmpeg** (untuk media core).

Terminal 1 — core:

```bash
cd src
cp .env.example .env        # sesuaikan: APP_PORT, APP_BASIC_AUTH, WHATSAPP_WEBHOOK, dll.
go run . rest               # REST API port 3000
```

Terminal 2 — dashboard:

```bash
cd dashboard
cp .env.example .env        # set WHATSAPP_API_URL, DASHBOARD_TZ, dll.
go mod tidy
go run .                    # http://localhost:8088
```

Build binary single-file:

```bash
# core
cd src && go build -o whatsapp && ./whatsapp rest

# dashboard (pure-Go, no CGO)
cd dashboard && CGO_ENABLED=0 go build -ldflags="-w -s" -o whatsapp-dashboard
```

Windows: ada helper `dashboard/start.bat`.

### Opsi C — MCP mode

```bash
cd src && go run . mcp      # http://localhost:8080
```

REST dan MCP **tidak bisa jalan bersamaan** dalam satu proses.

---

## Konfigurasi

Prioritas: **CLI flag > environment variable > `.env`**.

### Dashboard (`dashboard/.env`)

| Variable                | Default                  | Keterangan                                                                |
|-------------------------|--------------------------|---------------------------------------------------------------------------|
| `DASHBOARD_HOST`        | `0.0.0.0`                | Bind address.                                                             |
| `DASHBOARD_PORT`        | `8088`                   | Port HTTP.                                                                |
| `DASHBOARD_DB`          | `dashboard.db`           | Path file SQLite (dalam Docker default: `/data/dashboard.db`).            |
| `DASHBOARD_TZ`          | `Local`                  | Timezone default jadwal baru (mis. `Asia/Jakarta`).                       |
| `DASHBOARD_BASIC_AUTH`  | (kosong)                 | `user:pass` untuk proteksi UI dashboard. Kosong = terbuka.                |
| `WHATSAPP_API_URL`      | `http://localhost:3000`  | URL core REST API (di Docker: `http://whatsapp_go:3000`).                 |
| `WHATSAPP_API_USER`     | (kosong)                 | Basic auth user untuk core (jika `APP_BASIC_AUTH` di core diaktifkan).    |
| `WHATSAPP_API_PASSWORD` | (kosong)                 | Basic auth password untuk core.                                            |

### Core (`src/.env`)

Variable utama (full list lihat [`how-to-use.md`](./how-to-use.md)):

| Variable                       | Default                                       | Keterangan                                              |
|--------------------------------|-----------------------------------------------|---------------------------------------------------------|
| `APP_PORT` / `APP_HOST`        | `3000` / `0.0.0.0`                            | Port & bind address core.                               |
| `APP_DEBUG`                    | `false`                                       | Logging debug.                                          |
| `APP_OS`                       | `Chrome`                                      | Nama device yang tampil di mobile WhatsApp.             |
| `APP_BASIC_AUTH`               | -                                             | `user1:pass1,user2:pass2`.                              |
| `APP_BASE_PATH`                | -                                             | Subpath deploy (mis. `/gowa`).                          |
| `DB_URI`                       | `file:storages/whatsapp.db?_foreign_keys=on`  | SQLite default; bisa `postgres://...`.                  |
| `WHATSAPP_AUTO_REPLY`          | -                                             | Pesan auto-reply.                                       |
| `WHATSAPP_AUTO_MARK_READ`      | `false`                                       | Otomatis tandai pesan masuk sebagai dibaca.             |
| `WHATSAPP_AUTO_DOWNLOAD_MEDIA` | `true`                                        | Otomatis download media masuk.                          |
| `WHATSAPP_WEBHOOK`             | -                                             | URL webhook (boleh CSV multi-URL).                      |
| `WHATSAPP_WEBHOOK_SECRET`      | `secret`                                      | HMAC secret untuk header signature.                     |
| `WHATSAPP_WEBHOOK_EVENTS`      | -                                             | Filter event (kosong = semua).                          |
| `WHATSAPP_PRESENCE_ON_CONNECT` | `unavailable`                                 | `available` / `unavailable` / `none`.                   |
| `CHATWOOT_ENABLED`             | `false`                                       | Aktifkan Chatwoot sync; butuh `CHATWOOT_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`, `CHATWOOT_INBOX_ID`, `CHATWOOT_DEVICE_ID`. |
| `AI_REPLY_ENABLED`             | `false`                                       | Feature gate AI auto-reply. Saat `true`, **`AI_ENCRYPTION_KEY` wajib** diisi.        |
| `AI_ENCRYPTION_KEY`            | -                                             | 32-byte hex (64 chars) untuk AES-GCM enkripsi API key di SQLite. Generate: `openssl rand -hex 32`. **Backup terpisah** — kehilangan key = API key tersimpan unreadable. |
| `AI_MAX_KB_FILE_SIZE`          | `10485760`                                    | Max ukuran upload dokumen knowledgebase (bytes, default 10MB).                       |
| `AI_REQUEST_TIMEOUT_SEC`       | `10`                                          | Per-call timeout untuk LLM + embeddings (detik). Untuk Sumopod/provider lambat naikkan ke 30–60. |
| `AI_RATE_LIMIT_SECONDS`        | `3`                                           | Interval minimum (detik) antara reply AI per chat.                                   |
| `AI_VECTOR_DIMENSION`          | `1536`                                        | Dimensi embedding (default cocok untuk `text-embedding-3-small`).                    |

---

## REST API Dashboard

Semua di-prefix `/api`. Endpoint device adalah proxy ke core (otomatis menyisipkan `X-Device-Id`, basic auth, dll.).

| Method   | Path                          | Keterangan                                                  |
|----------|-------------------------------|-------------------------------------------------------------|
| `GET`    | `/api/_health`                | Probe versi build + URL upstream.                            |
| `GET`    | `/api/devices`                | List semua device.                                           |
| `POST`   | `/api/devices`                | Buat device baru. Body: `{"device_id":"alias"}`.             |
| `DELETE` | `/api/devices/:id`            | Hapus device.                                                |
| `GET`    | `/api/devices/:id/status`     | Status koneksi (connected/loggedIn/dll).                     |
| `GET`    | `/api/devices/:id/login`      | Mulai QR login; balikan `qr_link` sudah di-rewrite ke `/api/qr/...`. |
| `GET`    | `/api/devices/:id/login-code` | Login pakai kode telepon. Query: `phone=628xxx`.             |
| `POST`   | `/api/devices/:id/logout`     | Logout device.                                               |
| `POST`   | `/api/devices/:id/reconnect`  | Reconnect socket.                                            |
| `GET`    | `/api/qr/:filename`           | Proxy gambar QR PNG dari core.                               |
| `POST`   | `/api/send`                   | Kirim pesan sekarang (text/image/video/file/audio/location/link). |
| `GET`    | `/api/schedules`              | List jadwal.                                                 |
| `POST`   | `/api/schedules`              | Buat jadwal.                                                 |
| `GET`    | `/api/schedules/:id`          | Detail jadwal.                                               |
| `PUT`    | `/api/schedules/:id`          | Update jadwal.                                               |
| `DELETE` | `/api/schedules/:id`          | Hapus jadwal.                                                |
| `POST`   | `/api/schedules/:id/toggle`   | Enable/disable.                                              |
| `POST`   | `/api/schedules/:id/run`      | Eksekusi sekali sekarang (manual).                           |
| `GET`    | `/api/schedules/:id/logs`     | Log eksekusi per jadwal. Query: `?limit=50`.                 |
| `POST`   | `/api/schedules/preview`      | Preview N fire-time berikutnya (tanpa simpan). `?count=5`.    |
| `GET`    | `/api/logs`                   | Log eksekusi global terbaru. Query: `?limit=100`.            |
| `GET`    | `/api/aireply/config`         | Config AI per-device. API key dikembalikan dalam bentuk masked (`sk-v****Aw5A`). |
| `PUT`    | `/api/aireply/config`         | Simpan config. API key kosong = pertahankan yang tersimpan.   |
| `POST`   | `/api/aireply/config/test`    | Test koneksi provider; balas `{latency_ms, model_response}`.  |
| `POST`   | `/api/aireply/documents`      | Upload dokumen KB (`multipart/form-data` field `file`).       |
| `GET`    | `/api/aireply/documents`      | List dokumen KB beserta status (`processing`/`ready`/`failed`). |
| `DELETE` | `/api/aireply/documents/:id`  | Hapus dokumen + semua chunk-nya.                              |
| `POST`   | `/api/aireply/documents/reindex` | Re-embed semua chunk (pakai setelah ganti embed model).    |
| `GET`    | `/api/aireply/chat-settings`  | List chat opt-in per-device.                                  |
| `PUT`    | `/api/aireply/chat-settings/:chat_jid` | Toggle AI on/off untuk satu chat. Body `{"enabled":bool}`. |
| `GET`    | `/api/aireply/logs`           | Audit log eksekusi AI. Query: `?chat_jid=&status=&limit=50`.  |

Contoh payload `POST /api/schedules`:

```json
{
  "name": "Reminder rapat mingguan",
  "device_id": "6289605618749@s.whatsapp.net",
  "recipient": "120363xxxxxxxxxxxx@g.us",
  "message_type": "text",
  "message": "Halo tim, rapat jam 09:00.",
  "schedule_type": "weekly",
  "run_at": "2026-05-18T09:00",
  "cron_expr": "1,3,5",
  "timezone": "Asia/Jakarta",
  "enabled": true
}
```

Untuk REST API core (kirim langsung tanpa dashboard), lihat [`docs/openapi.yaml`](./docs/openapi.yaml).

---

## Tipe Jadwal

| Tipe      | Field yang dipakai                                   | Contoh                                                          |
|-----------|------------------------------------------------------|-----------------------------------------------------------------|
| `once`    | `run_at` (tanggal + jam)                             | Reminder satu kali 12 Mei 2026 jam 14:00.                       |
| `daily`   | `run_at` (jam-menit diambil)                         | Tiap hari jam 08:00.                                            |
| `weekly`  | `run_at` (jam-menit) + `cron_expr` CSV hari (0=Min)  | `cron_expr="1,3,5"` jam 09:00 → Senin/Rabu/Jumat.               |
| `monthly` | `run_at` (tanggal + jam)                             | Tiap tanggal 1 jam 07:00.                                       |
| `yearly`  | `run_at` (bulan + tanggal + jam)                     | Setiap 17 Agustus jam 10:00.                                    |
| `cron`    | `cron_expr` 5-field                                  | `0 9 * * 1-5` → jam 09:00 setiap weekday.                       |

Format cron: 5 field `menit jam hari-bulan bulan hari-pekan` (parser `robfig/cron/v3`, hari-pekan `0-6` dengan `0=Minggu`).

Tipe pesan yang valid: `text`, `image`, `video`, `file`, `audio`, `location`, `link`. Field wajib berbeda per tipe (mis. `media_url` untuk media, `latitude`+`longitude` untuk location, `link_url` untuk link). Validasi penuh ada di `dashboard/internal/api/handlers.go`.

---

## Catatan Penting

- **Jangan modifikasi `src/`** kecuali memang perlu fork penuh — dashboard sengaja didesain sebagai overlay sehingga `git pull` upstream aman.
- **Device ID vs JID** (penting saat integrasi ke API): `device_id` di header bisa alias ("my-device") atau JID; saat menyimpan/lookup data chat selalu pakai JID *tanpa* device-number (`ToNonAD()`). Detail lengkap di [`CLAUDE.md`](./CLAUDE.md).
- **FFmpeg + libwebp** diperlukan oleh core untuk konversi media & sticker. Image Docker bawaan sudah menyertakan keduanya.
- **Database**: core default `storages/whatsapp.db` (SQLite); set `DB_URI=postgres://...` untuk PostgreSQL. Dashboard selalu pakai SQLite pure-Go lokal — tidak ada CGO, binary jalan di Alpine tanpa libc tambahan.
- **Webhook payload v8+** menyertakan `device_id` top-level (lihat [`docs/webhook-payload.md`](./docs/webhook-payload.md)).
- **AI Encryption Key**: kalau pakai AI Reply, `AI_ENCRYPTION_KEY` di `src/.env` adalah satu-satunya cara men-decrypt API key provider yang tersimpan di SQLite. **Backup terpisah** (mis. password manager). Kalau hilang/berubah, semua API key tersimpan jadi unreadable dan harus di-input ulang via UI. Untuk dokumentasi lengkap fitur AI Reply lihat [`docs/PRD-ai-auto-reply.md`](./docs/PRD-ai-auto-reply.md).
- **Cache**: dashboard mengembalikan `Cache-Control: no-store` untuk seluruh static asset, supaya UI selalu pakai versi terbaru setelah rebuild Docker image.

---

## Lisensi & Atribusi

- Core (`src/`) adalah karya [@aldinokemal](https://github.com/aldinokemal) — lihat [`LICENCE.txt`](./LICENCE.txt) dan repo upstream [`aldinokemal/go-whatsapp-web-multidevice`](https://github.com/aldinokemal/go-whatsapp-web-multidevice). Dukung lewat [Patreon](https://www.patreon.com/c/aldinokemal) jika kamu pakai untuk produksi.
- Dashboard (`dashboard/`) ditulis ulang dari nol sebagai overlay di repo ini; mengikuti lisensi yang sama (MIT).
