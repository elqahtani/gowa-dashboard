# PRD: AI Auto-Reply + RAG Knowledgebase

**Status**: Draft (Phase 1 MVP)
**Owner**: kerupuk.lovers@gmail.com
**Tanggal**: 2026-05-16
**Versi**: 0.1

---

## 1. Latar Belakang

`gowa-dashboard` saat ini sudah punya fitur **auto-reply** sederhana di
`src/infrastructure/whatsapp/auto_reply.go` yang hanya mengirim **static text**
ke semua DM non-grup. Untuk use-case customer service / sales produk, ini
tidak cukup — admin tetap harus reply manual untuk pertanyaan kontekstual
(FAQ produk, harga, cara order, dst).

PRD ini mendefinisikan upgrade ke **AI-powered auto-reply** dengan:

- **Retrieval-Augmented Generation (RAG)** dari dokumen yang di-upload user
  (PDF / TXT / DOCX) sebagai single source of truth.
- **Multi-provider AI** — Claude (Anthropic), atau OpenAI-compatible
  (OpenRouter, Sumopod, DeepSeek, Groq, Ollama, OpenAI).
- **Guardrail** — AI hanya boleh menjawab dari konteks chat & dokumen.
  Pertanyaan di luar scope ditolak dengan template message yang sopan.
- **Prompt style** configurable supaya tone WA agent bisa disesuaikan
  (formal, santai, technical support, custom).
- **Opt-in per chat** — admin aktifkan AI hanya untuk chat tertentu.
- **Per-device scoping** — KB dan config terisolasi per WhatsApp account
  (konsisten dengan multi-device architecture existing).

## 2. Problem Statement

> Admin produk harus menjawab pertanyaan customer yang berulang di WA
> setiap hari. Pertanyaan-pertanyaan ini sebenarnya jawabannya sudah ada
> di brosur PDF / katalog produk / FAQ. Tidak ada cara untuk
> meng-automate jawaban tersebut tanpa kehilangan personal touch dan
> tanpa risiko AI mengarang jawaban di luar pengetahuan produk.

## 3. Goals

| ID  | Goal                                                                 | Metric                                              |
|-----|----------------------------------------------------------------------|-----------------------------------------------------|
| G1  | Reduce manual reply effort untuk FAQ produk                          | ≥ 60% incoming pertanyaan FAQ auto-handled          |
| G2  | AI hanya jawab dari knowledgebase + chat context (no hallucination)  | < 5% reply yang di luar scope KB (manual audit)     |
| G3  | User bisa upload/manage KB tanpa edit kode / restart server          | Upload PDF 5MB → ready < 30 detik                   |
| G4  | Style WA agent (formal/santai/dll) bisa diubah dari UI               | Style change → reply berikutnya pakai style baru    |
| G5  | Latency reply yang acceptable                                        | P95 latency reply < 5 detik (retrieval + LLM)       |

## 4. Non-Goals (MVP)

- Voice message transcription input
- Image / multimodal understanding (gambar dari customer)
- Generate gambar / file balasan
- Multi-turn agent dengan tool calling
- Training / fine-tune model
- PostgreSQL + pgvector support (MVP fokus SQLite + sqlite-vec)
- Per-chat custom prompt override (cuma device-level dulu)
- Webhook event `ai.reply.sent` untuk integrasi eksternal

## 5. User Personas

- **Admin/Owner produk**: punya WA business, sudah pakai gowa-dashboard
  untuk handle multiple device. Ingin reduce beban reply FAQ.
- **Customer**: end-user yang chat di WA, expect jawaban cepat &
  konsisten dengan info produk.

## 6. User Stories

### US-1 — Upload knowledgebase
> Sebagai admin, saya upload PDF katalog produk lewat UI →
> sistem otomatis extract teks, chunk, embed via AI provider,
> simpan ke vector store. Status berubah `processing` → `ready`.

### US-2 — Konfigurasi AI provider
> Sebagai admin, saya pilih provider (Anthropic / OpenAI-compatible),
> masukin API key + model name + (opsional) base URL custom,
> save config per-device.

### US-3 — Pilih prompt style
> Sebagai admin, saya pilih preset style (`Customer Service Formal` /
> `Sales Casual` / `Technical Support` / `Custom`) yang menentukan
> tone & bahasa WA agent. Untuk `Custom` saya bisa tulis system prompt
> sendiri.

### US-4 — Aktifkan AI per chat
> Sebagai admin, saya toggle "AI auto-reply" ON di chat customer X →
> pesan masuk dari X otomatis dibalas AI grounded di KB. Customer Y
> yang OFF tetap saya handle manual.

### US-5 — Audit log
> Sebagai admin, saya lihat halaman audit log: pertanyaan customer,
> chunks yang ter-retrieve, jawaban AI, latency, status (success /
> out_of_scope / error). Bisa filter by chat & status.

### US-6 — Guardrail out-of-scope
> Saat customer tanya hal yang tidak ada di KB ("besok cuaca gimana?"),
> AI nolak halus dengan template message yang sudah dikonfigurasi
> (default: "Maaf, saya hanya bisa bantu seputar topik yang ada di
> knowledgebase kami.").

### US-7 — Hot-reload config
> Sebagai admin, perubahan config (prompt style, threshold, max tokens,
> dst.) langsung berlaku untuk pesan berikutnya tanpa restart server.

## 7. Functional Requirements

### FR-1 — Provider Abstraction
Interface `IAIProvider` dengan 2 implementasi:

- `AnthropicProvider` — pakai SDK resmi `github.com/anthropics/anthropic-sdk-go`
  (chat only, embeddings fallback ke OpenAI-compatible).
- `OpenAICompatibleProvider` — pakai `github.com/sashabaranov/go-openai`
  dengan custom `BaseURL`. Cover **OpenAI**, **OpenRouter**, **Sumopod**,
  **DeepSeek**, **Groq**, **Ollama** — semua expose `/v1/chat/completions`
  + `/v1/embeddings`.

Config user pilih `provider` + `model` + `api_key` + `base_url` (optional)
+ (kalau provider Anthropic) `embed_provider` / `embed_model` / `embed_api_key`
/ `embed_base_url` untuk embeddings.

### FR-2 — Knowledgebase Upload
- Endpoint `POST /aireply/documents` multipart form, field `file`.
- Format yang diterima: PDF, TXT, DOCX.
- Max file size default 10MB (configurable via env `AI_MAX_KB_FILE_SIZE`).
- Validasi: extension + MIME type + size.
- Response: `{ id, filename, status: "processing" }`.
- Background goroutine parse → chunk → embed → insert ke vec store.
- Status final: `ready` atau `failed` (dengan `error_message`).

### FR-3 — Document Parsing & Chunking
- **PDF** via `github.com/ledongthuc/pdf` (pure Go).
- **DOCX** via `github.com/nguyenthenguyen/docx`.
- **TXT** plain read.
- Validasi teks hasil extract panjangnya ≥ 50 chars; kalau gagal → status
  `failed` dengan pesan jelas (kemungkinan scanned PDF, suggest convert
  manual ke TXT).
- Chunking: target ~500 tokens per chunk dengan overlap ~50 tokens.
  Approximation token = panjang chars / 4. Split di natural boundary
  (paragraph break → sentence break → word break).

### FR-4 — RAG Retrieval
- Pada pesan masuk yang lolos skip rules + chat enabled + rate limit pass:
  1. Embed query (pesan customer) via embed provider.
  2. Cosine similarity search top-K (default 4) di `kb_chunks_vec` scoped
     `device_id`.
  3. Kalau top-1 similarity < `retrieval_threshold` (default 0.3) → langsung
     kirim `out_of_scope_message`, log status `out_of_scope`. **Skip LLM
     call** untuk hemat token.
  4. Kalau pass → lanjut build prompt.

### FR-5 — Per-Chat Opt-In
- Table `ai_chat_settings(device_id, chat_jid, enabled, updated_at)`.
- Default `enabled = false`.
- UI menyediakan toggle per chat (akan diintegrasikan ke chat list
  existing di Phase berikutnya; MVP: dedicated page list chat + toggle).
- Endpoint `PUT /aireply/chat-settings/:chat_jid` body `{ enabled: bool }`.

### FR-6 — Prompt Style & Config
- Table `ai_config(device_id PK, provider, model, api_key_encrypted, ...)`
  (lihat skema lengkap di Section 11).
- Style presets di kode:
  - `customer_service_formal`
  - `sales_casual`
  - `technical_support`
  - `custom` (user fill sendiri)
- Final system prompt = `style_text + guardrail_instruction + <context>`.

### FR-7 — Guardrail
**Layer 1 — Pre-LLM**: threshold check (FR-4 step 3).

**Layer 2 — System prompt**: hardcoded di prompt builder:
> "Jawab HANYA berdasarkan informasi di dalam tag `<context>` dan
> riwayat chat. Kalau informasi yang ditanyakan tidak ada di context,
> balas persis dengan: `{out_of_scope_message}`. JANGAN mengarang fakta,
> JANGAN memakai pengetahuan umum di luar context."

**Layer 3 (Phase 2)**: post-LLM regex check untuk phrase hallucination.

### FR-8 — Chat History Context
- Inject 5 pesan terakhir dari chat tsb (mix incoming + outgoing) ke
  prompt sebagai conversation history.
- Pakai `chatstorage.GetMessages(MessageFilter{DeviceID, ChatJID, Limit: 5})`
  yang sudah ada.

### FR-9 — Skip Rules
Reuse logic dari `auto_reply.go:27-47`:
- Skip grup, broadcast, `status@broadcast`, self-messages
  (`IsFromMe = true`).
- Skip pesan non-text (image, audio, video, document, sticker, location,
  contact, reaction).
- Hanya server `@s.whatsapp.net` & `@lid` yang diproses.

### FR-10 — Rate Limiting
- In-memory `sync.Map` per `chat_jid`.
- Max 1 AI reply per chat per N detik (default 3, configurable via env
  `AI_RATE_LIMIT_SECONDS`).
- Reply yang ke-rate-limit dilog status `rate_limited`, **tidak** kirim
  message ke customer.

### FR-11 — Audit Log
- Table `ai_reply_logs` dengan field: `id, device_id, chat_jid, query,
  retrieved_chunk_ids (JSON), response, latency_ms, tokens_in,
  tokens_out, status, error_message, created_at`.
- Status enum: `success` | `out_of_scope` | `error` | `rate_limited`.
- Endpoint `GET /aireply/logs?chat_jid=&status=&limit=` dengan default
  limit 50.

### FR-12 — Backward Compatibility
- Static `auto_reply.go` existing tetap berfungsi.
- Kalau AI auto-reply ON untuk chat tertentu → AI handle, static skip.
- Kalau AI OFF / config tidak ada / error → fallback ke static (kalau
  static dikonfigurasi).

## 8. Non-Functional Requirements

| ID    | Requirement                                                                          |
|-------|--------------------------------------------------------------------------------------|
| NFR-1 | P95 latency reply < 5 detik (retrieval + LLM). Async send, jangan block event loop.  |
| NFR-2 | API key encrypted at rest (AES-GCM, key dari env `AI_ENCRYPTION_KEY`). Tidak pernah log API key. |
| NFR-3 | Max tokens reply default 500. Total prompt cap 4000 tokens (truncate KB chunks).    |
| NFR-4 | Semua DB query scoped `device_id` (multi-device isolation).                          |
| NFR-5 | Provider error / timeout (default 10s) → skip reply diam-diam, log ke audit. JANGAN kirim error ke customer. |
| NFR-6 | Migrations append-only ke `getMigrations()`. Backward compatible.                    |
| NFR-7 | Feature gate: kalau `AI_REPLY_ENABLED=false`, hook handler langsung return.          |

## 9. UI / UX

### Halaman baru "AI Reply" (nav tab di dashboard)

**Sub-tab 1 — Configuration**
- Provider dropdown (Anthropic / OpenAI-compatible)
- Model text input (placeholder: `claude-sonnet-4-6`, `gpt-4o-mini`, `anthropic/claude-3.5-sonnet`)
- API Key password input (write-only; current value masked saat fetch)
- Base URL text input (optional, untuk OpenAI-compatible custom endpoint)
- Embed provider section (collapsible, default hidden kalau provider = openai_compatible)
- Style preset dropdown + textarea custom prompt (visible kalau pilih custom)
- Sliders: `max_tokens`, `temperature`, `top_k`, `retrieval_threshold`
- Out-of-scope message textarea
- Button: **Save**, **Test Connection**

**Sub-tab 2 — Knowledgebase**
- Upload area (drag-drop) + format hint
- Table: filename, size, chunks, status, created_at, [delete]
- Button **Reindex All** (untuk ganti embed model)

**Sub-tab 3 — Active Chats**
- Table list chat yang punya setting AI: chat_jid, last_message_time,
  toggle on/off, [delete setting]
- Search box untuk find chat
- Toggle untuk enable AI di chat baru via JID input

**Sub-tab 4 — Audit Logs**
- Table: timestamp, chat_jid, query (truncated), response (truncated),
  status, latency, tokens
- Filter: chat_jid, status, date range
- Click row → expand detail

## 10. Konfigurasi & Environment Variables

| Env Var                | Default                           | Deskripsi                                |
|------------------------|-----------------------------------|------------------------------------------|
| `AI_REPLY_ENABLED`     | `false`                           | Feature gate global                      |
| `AI_ENCRYPTION_KEY`    | (empty, wajib kalau enabled)      | 32-byte hex key untuk AES-GCM            |
| `AI_MAX_KB_FILE_SIZE`  | `10485760` (10MB)                 | Max upload size dalam bytes              |
| `AI_REQUEST_TIMEOUT`   | `10` (detik)                      | Timeout per LLM call                     |
| `AI_RATE_LIMIT_SECONDS`| `3`                               | Min interval antar AI reply per chat     |
| `AI_VECTOR_DIMENSION`  | `1536`                            | Dim embedding (text-embedding-3-small)   |

## 11. Data Model (Migrations 17-22)

Lihat plan file `cozy-orbiting-otter.md` untuk full SQL DDL. Ringkasan
tabel:

- **`ai_config`** — config per-device (provider, model, encrypted keys,
  style, threshold, dst)
- **`kb_documents`** — metadata upload (filename, size, chunk_count,
  status)
- **`kb_chunks`** — chunk teks dengan FK ke `kb_documents`
- **`kb_chunks_vec`** — virtual table sqlite-vec (chunk_id PK,
  embedding float[1536])
- **`ai_chat_settings`** — toggle per (device, chat_jid)
- **`ai_reply_logs`** — audit log

Semua tabel scoped `device_id` (kecuali `kb_chunks_vec` yang punya kolom
`device_id` sebagai filter di WHERE clause).

## 12. API Endpoints

| Method | Path                                  | Deskripsi                            |
|--------|---------------------------------------|--------------------------------------|
| GET    | `/aireply/config`                     | Ambil config (api_key masked)        |
| PUT    | `/aireply/config`                     | Save / update config                 |
| POST   | `/aireply/config/test`                | Test koneksi ke provider             |
| POST   | `/aireply/documents`                  | Upload KB document (multipart)       |
| GET    | `/aireply/documents`                  | List KB documents                    |
| DELETE | `/aireply/documents/:id`              | Delete doc + chunks + vec            |
| POST   | `/aireply/documents/reindex`          | Re-embed semua dokumen               |
| GET    | `/aireply/chat-settings`              | List chat yang ada setting           |
| PUT    | `/aireply/chat-settings/:chat_jid`    | Toggle on/off per chat               |
| GET    | `/aireply/logs`                       | Audit log (filter + paginate)        |

## 13. Risiko & Mitigasi

| Risiko                                              | Mitigasi                                                 |
|-----------------------------------------------------|----------------------------------------------------------|
| sqlite-vec extension gagal load di env user         | Detect saat startup, log error jelas, disable AI gracefully (jangan crash core feature). |
| PDF scanned → extraction garbage                    | Validate min 50 chars text per doc, status=failed + pesan suggest convert manual. |
| `AI_ENCRYPTION_KEY` kosong saat user enable AI      | Startup check + REST save reject dengan pesan eksplisit. |
| AI balas AI (loop antar bot)                        | Skip rule `IsFromMe` + rate limit + future blocklist JID.|
| Cost API meledak                                    | `max_tokens=500` + threshold pre-filter + chunk truncation cap 4000 tokens prompt total. |
| Ganti embed model → dim mismatch                    | Endpoint `POST /documents/reindex` untuk re-embed semua. |
| Customer kirim prompt injection ("ignore previous instruction") | System prompt strict + isolasi context via XML tags `<context>` + max_tokens cap reply length. |

## 14. Success Metrics (Post-Launch)

- ≥ 80% AI reply latency < 5 detik (P95)
- < 5% audit logs `status = error`
- ≥ 95% KB upload success rate untuk file < 10MB
- < 5% reply yang diluar scope KB (manual audit sample 100 reply)
- Subjective: admin lapor reduce manual reply ≥ 60% untuk chat yang AI-enabled

## 15. Rollout Plan

1. **Phase 1 (MVP)** — fitur ini. Single-device beta dengan dokumen
   sederhana dulu.
2. **Phase 2** — PostgreSQL + pgvector, multimodal (image input),
   per-chat custom prompt, MCP tools lengkap.
3. **Phase 3** — Webhook event `ai.reply.sent`, A/B testing prompts,
   analytics dashboard.

## 16. Open Questions

- Apakah perlu blocklist nomor (selain `IsFromMe`) untuk cegah bot loop?
  → Phase 2.
- Apakah perlu human-in-the-loop approval mode (AI draft → admin approve
  baru kirim)? → Phase 2 / Phase 3.
- Multi-language support — system prompt saat ini Indonesian; perlu
  auto-detect? → MVP user manual set via `style_preset = custom`.

---

## Referensi

- Plan implementasi: `/Users/mini/.claude/plans/cozy-orbiting-otter.md`
- Codebase docs: `CLAUDE.md` di root project
- WhatsApp integration: `src/infrastructure/whatsapp/`
- Auto-reply existing: `src/infrastructure/whatsapp/auto_reply.go`
