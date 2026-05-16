# Go WhatsApp Web Multi-Device + Dashboard

This repo bundles **two independent Go applications**:

1. **`src/`** — gowa core (upstream [`aldinokemal/go-whatsapp-web-multidevice`](https://github.com/aldinokemal/go-whatsapp-web-multidevice)). WhatsApp Web API server with REST + MCP modes. Multi-device via whatsmeow. **Treat as upstream** — minimise changes here to keep `git pull` low-conflict.
2. **`dashboard/`** — companion app (port 8088). Separate Go binary that proxies core's REST API and adds a Vue 3 SPA for multi-device management, message scheduling (one-shot/daily/weekly/monthly/yearly/cron via robfig/cron), and a UI for the AI Reply feature. Talks to core over HTTP. State (schedules + logs) lives in its own SQLite (`modernc.org/sqlite`, pure-Go, no CGO).

Run them together with `docker compose -f docker-compose.full.yml up -d --build`.

## Structure

```
src/                              # ── gowa CORE (upstream-aligned) ──
├── main.go                       # Entry point; embeds views/ via go:embed
├── cmd/                          # Cobra CLI: rest, mcp subcommands + root config
├── config/settings.go            # All config vars (Viper-bound) — includes 6 AI_* gates
├── domains/                      # Contracts only: interfaces + DTOs
│   └── aireply/                  # AI Reply: AIConfig, KBDocument, ReplyLog, IService, etc.
├── infrastructure/
│   ├── whatsapp/                 # WhatsApp protocol layer
│   │   ├── ai_reply.go           # Bridges events → IAIReplyHandler (with LID-norm + detached ctx + presence)
│   │   ├── jid_utils.go          # NormalizeJIDFromLID
│   │   └── event_*.go            # One file per event type, registered in event_handler.go
│   ├── chatstorage/              # SQLite/PostgreSQL persistence (chats, messages)
│   ├── chatwoot/                 # Chatwoot CRM bidirectional sync
│   └── aireply/                  # AI Reply: SQLite repo + sqlite-vec virtual table store
├── usecase/
│   ├── ...                       # Business logic (1:1 with domains)
│   └── aireply/                  # Service orchestrator + chunker, parser, prompt builder,
│                                 # AES-GCM crypto, rate limiter, Anthropic + OpenAI providers
├── validations/                  # ozzo-validation input checks + tests
├── ui/
│   ├── rest/                     # Fiber HTTP handlers + middleware (aireply.go = 10 endpoints)
│   └── mcp/                      # MCP server handlers
├── views/                        # Vue.js 3 components (Semantic UI, plain JS) — includes AI*.js
├── pkg/                          # Shared helpers + error types
├── statics/                      # Runtime media + QR codes
└── storages/                     # Runtime SQLite DBs (chats, messages, ai_config, kb_*)

dashboard/                        # ── DASHBOARD COMPANION (separate binary) ──
├── main.go                       # Fiber app, port 8088, embeds web/ via go:embed
├── internal/
│   ├── api/handlers.go           # /api/* routes (devices, send, schedules, aireply proxy)
│   ├── wa/client.go              # Thin HTTP client to core (forwards X-Device-Id)
│   ├── scheduler/scheduler.go    # robfig/cron + time.Timer (one-shot vs recurring)
│   ├── store/store.go            # SQLite for schedules + execution logs
│   └── config/config.go          # godotenv loader
└── web/index.html                # Single-file Vue 3 SPA (5 tabs: Devices, Send, Schedules,
                                  # Logs, AI Reply). AI Reply tab proxies to core /aireply/*.
```

## Commands

```bash
# Core (src/)
cd src && go run . rest                   # REST API (port 3000)
cd src && go run . mcp                    # MCP server (port 8080)
cd src && go build -o whatsapp            # Build binary
cd src && go test ./...                   # Run all tests
cd src && go vet ./...                    # Static analysis

# Dashboard (dashboard/)
cd dashboard && go run .                  # Dashboard (port 8088)
cd dashboard && CGO_ENABLED=0 go build -ldflags="-w -s" -o whatsapp-dashboard

# Combined Docker (recommended for local dev)
docker compose -f docker-compose.full.yml up -d --build
docker compose -f docker-compose.full.yml up -d --build dashboard   # rebuild dashboard only
docker compose -f docker-compose.full.yml up -d --build whatsapp_go # rebuild core only
```

## Where to Look

### Core (`src/`)

| Task | Location | Notes |
|------|----------|-------|
| Add message type | `domains/send/`, `usecase/send.go`, `validations/send_validation.go` | 3-file pattern |
| Add API endpoint | `ui/rest/`, `usecase/`, `domains/` | Handler → usecase → domain |
| Add MCP tool | `ui/mcp/` | Mirrors REST; `query.go` for read ops |
| Handle WA event | `infrastructure/whatsapp/event_*.go` | Register in `event_handler.go` switch |
| Add DB migration | `infrastructure/chatstorage/sqlite_repository.go` → `getMigrations()` | Append only — never insert in middle |
| Add Vue component | `views/components/` | Plain JS, no .vue SFC |
| Device management | `infrastructure/whatsapp/device_manager.go` | Central orchestrator |
| Chatwoot integration | `infrastructure/chatwoot/` | `client.go` (API) + `sync.go` (sync) |
| AI Reply config / KB / chat-toggle / logs | `usecase/aireply/service.go`, `infrastructure/aireply/`, `ui/rest/aireply.go` | 10 endpoints under `/aireply/*`, device-scoped via `X-Device-Id` |
| AI provider (Claude / OpenAI-compat) | `usecase/aireply/provider_anthropic.go`, `provider_openai.go` | Implements `IAIProvider` |
| AI Reply event bridge | `infrastructure/whatsapp/ai_reply.go` | Runs **before** static auto-reply in `event_message_handler.go`; LID-normalises chat/sender JIDs; uses detached ctx |
| LID resolution | `infrastructure/whatsapp/jid_utils.go` (`NormalizeJIDFromLID`) | Required before any per-chat DB lookup |
| CLI flags / config | `cmd/root.go` | Viper+Cobra, `.env` loading; AI flags wired in `initFlags()` + `initEnvConfig()` |
| Shared helpers | `pkg/utils/whatsapp.go`, `general.go` | JID, media, phone formatting |

### Dashboard (`dashboard/`)

| Task | Location | Notes |
|------|----------|-------|
| Add dashboard endpoint | `internal/api/handlers.go` + `internal/wa/client.go` | Handler proxies via thin client |
| Add dashboard UI tab | `web/index.html` | Single Vue 3 app; add tab in nav + `<div v-if="tab === '...'">`, then data + methods |
| Schedule logic | `internal/scheduler/scheduler.go` | `robfig/cron` for recurring, `time.AfterFunc` for one-shot |
| Schedule DB | `internal/store/store.go` | One file, includes migrations inline |
| Proxy QR image | `handlers.go:qrImage` | Rewrites core's `qr_link` to `/api/qr/:filename` so browser doesn't need direct core access |

## Critical: Device ID vs JID

Two distinct identifiers — confusing them causes silent data bugs:

- **Device ID** (`instance.ID()`): User alias or UUID (e.g., `"my-device"`)
- **JID** (`instance.JID()`): WhatsApp JID (e.g., `"6289605618749@s.whatsapp.net"`)

The `chats`/`messages` tables store device_id as the **JID without device number**:
```go
deviceID := client.Store.ID.ToNonAD().String()  // ✅ "6289605618749@s.whatsapp.net"
// NOT instance.ID()  // ❌ may return alias or "6289605618749:11@s.whatsapp.net"
```

## Critical: AI Reply gating

`HandleIncoming` returns `false` (skips AI) silently if **any** of these:
1. `config.AIReplyEnabled` is false (env: `AI_REPLY_ENABLED`)
2. `deviceID` empty, or `text` empty after trim
3. No chat-setting row for this chat, or `enabled=false` — opt-in is **per chat JID**, there is no global "AI for all" toggle
4. Rate limit (default 3s/chat) hit — logged as `rate_limited`, ownership claimed (static fallback also skipped)
5. No `ai_config` row for device

Guardrail (`out_of_scope` template) only fires when there ARE KB chunks but query doesn't match threshold. **Empty KB auto-bypasses guardrail** so users without uploaded docs still get LLM answers (see `service.go` near the guardrailActive check). Pre-LLM `presence("composing")` is fired and refreshed every 10s; `presence("paused")` on send.

## Conventions

- **Clean Architecture**: `domains/` → `usecase/` → `ui/` (never reverse)
- **1:1 mapping**: Each domain has matching usecase, validation, and UI handler
- **Device scoping**: All chat/message DB queries must include `device_id`
- **LID normalization**: WhatsApp sends `@lid` JIDs — call `NormalizeJIDFromLID()` before DB ops
- **Optional booleans**: Use `*bool` for optional filter params (nil = not set)
- **Config priority**: CLI flags > env vars > `.env` file
- **Wrapper pattern**: `IChatStorageRepository` has two wrappers that inject device_id:
  - `infrastructure/whatsapp/chatstorage_wrapper.go` (for event handlers)
  - `infrastructure/chatstorage/device_repository.go` (for usecase layer)

## Anti-Patterns

- **Never** query chats/messages without device_id scoping
- **Never** use raw event JIDs for DB lookups without `NormalizeJIDFromLID` (events arrive as `@lid`; stored toggles/configs use `@s.whatsapp.net` form — silent mismatch is the failure mode)
- **Never** add `IChatStorageRepository` methods without updating both wrapper files
- **Never** insert migrations in the middle — always append to `getMigrations()`
- **Never** put business logic in domain packages — they define contracts only
- **Never** remove the Device == 0 check in receipt forwarding (prevents duplicate webhooks)
- **Never** propagate the whatsmeow event ctx to long work (LLM, embeddings, big DB scans). It expires fast and cancels mid-flight. Use `context.WithTimeout(context.Background(), ...)` — see `ai_reply.go` for the pattern.
- **Never** bind `nil []byte` to NOT NULL secret columns (e.g. `api_key_encrypted`). SQL driver sends NULL → constraint fires **before** ON CONFLICT UPDATE can run its preserve-existing CASE. Normalise `nil` → `[]byte{}` at the repo boundary.
- **Never** trust Fiber 2.52 `c.Params()` to URL-decode path params. `%40` stays as `%40`. Call `url.PathUnescape()` before use, or call core with `@` raw.
- **Never** modify `src/` for things that can live in `dashboard/`. The dashboard is the overlay; `src/` should stay close to upstream to keep `git pull` painless. Net new features that need event-handler hooks (like AI Reply) are the exception.

## Testing

- Standard Go testing + `testify/assert` + `testify/suite`
- Table-driven tests throughout
- `httptest` for HTTP testing, `fiber.Test()` for middleware
- Tests colocated with source: `*_test.go` next to implementation
- Run: `cd src && go test ./...`

## Key Dependencies

| Package | Role |
|---------|------|
| `go.mau.fi/whatsmeow` | WhatsApp Web protocol |
| `github.com/gofiber/fiber/v2` | REST framework (both core & dashboard) |
| `github.com/mark3labs/mcp-go` | MCP server (v0.45.0) |
| `github.com/spf13/cobra` + `viper` | CLI + config |
| `github.com/go-ozzo/ozzo-validation/v4` | Input validation |
| `github.com/mattn/go-sqlite3` / `github.com/lib/pq` | Core: SQLite (CGO) / PostgreSQL |
| `modernc.org/sqlite` | Dashboard: pure-Go SQLite, no CGO needed |
| `asg017/sqlite-vec` (via core SQLite) | Vector search for KB chunks (AI Reply) |
| `github.com/robfig/cron/v3` | Dashboard schedule engine |

## Notes

- **Go 1.25.0+** required (`src/go.mod`)
- REST and MCP modes cannot run simultaneously (whatsmeow limitation)
- FFmpeg required for media processing (`ConvertToJPEG`, `ConvertToMP4`)
- HTML/JS assets embedded in binary via `go:embed` (both core and dashboard)
- Database: core SQLite default, PostgreSQL via `DB_URI`; dashboard always SQLite (`./dashboard/data/dashboard.db` in Docker)
- Docker: multi-stage alpine, non-root users (core `gowauser` uid 20001, dashboard `dashuser` uid 20001)
- CI: GitHub Actions → GoReleaser (Linux/Windows/macOS) + multi-arch Docker (GHCR + Docker Hub)
- Hot reload: `.air.toml` configured (excludes `statics/`, `storages/`)
- `status@broadcast` chat always returns name "Status" regardless of other naming
- **AI Encryption Key**: `AI_ENCRYPTION_KEY` (32-byte hex) is the only way to decrypt API keys stored in `ai_config.api_key_encrypted`. **Treat like a password**; losing it = re-enter all provider API keys. Generate with `openssl rand -hex 32`.
- **Sumopod & similar OpenAI-compat providers** can be slow on cold start — set `AI_REQUEST_TIMEOUT_SEC=60` or higher.
- **Dashboard mirrors core's AI Reply** via `/api/aireply/*` (10 endpoints, all `X-Device-Id` scoped). Settings live in core's DB; dashboard is just a UI proxy. Don't add AI state to dashboard's SQLite.
- **Two distinct `.env` files** — `src/.env` (core) and `dashboard/.env` (dashboard). Both have `.env.example` templates that must stay in sync with the real ones.
