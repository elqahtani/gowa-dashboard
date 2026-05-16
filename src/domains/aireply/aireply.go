// Package aireply defines DTOs and interfaces for the AI-powered auto-reply
// feature with RAG knowledgebase support. Implementations live in
// usecase/aireply and infrastructure/aireply.
package aireply

import (
	"context"
	"mime/multipart"
	"time"
)

// Provider identifiers.
const (
	ProviderAnthropic        = "anthropic"
	ProviderOpenAICompatible = "openai_compatible"
)

// Style preset identifiers.
const (
	StyleCustomerServiceFormal = "customer_service_formal"
	StyleSalesCasual           = "sales_casual"
	StyleTechnicalSupport      = "technical_support"
	StyleCustom                = "custom"
)

// Document statuses.
const (
	DocStatusProcessing = "processing"
	DocStatusReady      = "ready"
	DocStatusFailed     = "failed"
)

// Reply log statuses.
const (
	LogStatusSuccess     = "success"
	LogStatusOutOfScope  = "out_of_scope"
	LogStatusError       = "error"
	LogStatusRateLimited = "rate_limited"
)

// Default out-of-scope template (Bahasa Indonesia).
const DefaultOutOfScopeMessage = "Maaf, saya hanya bisa bantu seputar topik yang ada di knowledgebase kami."

// AIConfig holds per-device AI provider + behaviour configuration.
type AIConfig struct {
	DeviceID             string    `json:"device_id"`
	Provider             string    `json:"provider"`
	Model                string    `json:"model"`
	APIKey               string    `json:"api_key,omitempty"` // plaintext only at the boundary
	BaseURL              string    `json:"base_url"`
	EmbedProvider        string    `json:"embed_provider"`
	EmbedModel           string    `json:"embed_model"`
	EmbedAPIKey          string    `json:"embed_api_key,omitempty"`
	EmbedBaseURL         string    `json:"embed_base_url"`
	SystemPrompt         string    `json:"system_prompt"`
	StylePreset          string    `json:"style_preset"`
	MaxTokens            int       `json:"max_tokens"`
	Temperature          float64   `json:"temperature"`
	TopK                 int       `json:"top_k"`
	RetrievalThreshold   float64   `json:"retrieval_threshold"`
	GuardrailEnabled     bool      `json:"guardrail_enabled"`
	OutOfScopeMessage    string    `json:"out_of_scope_message"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AIConfigRequest is the wire shape for PUT /aireply/config.
type AIConfigRequest struct {
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	APIKey             string  `json:"api_key"`
	BaseURL            string  `json:"base_url"`
	EmbedProvider      string  `json:"embed_provider"`
	EmbedModel         string  `json:"embed_model"`
	EmbedAPIKey        string  `json:"embed_api_key"`
	EmbedBaseURL       string  `json:"embed_base_url"`
	SystemPrompt       string  `json:"system_prompt"`
	StylePreset        string  `json:"style_preset"`
	MaxTokens          int     `json:"max_tokens"`
	Temperature        float64 `json:"temperature"`
	TopK               int     `json:"top_k"`
	RetrievalThreshold float64 `json:"retrieval_threshold"`
	GuardrailEnabled   bool    `json:"guardrail_enabled"`
	OutOfScopeMessage  string  `json:"out_of_scope_message"`
}

// KBDocument represents an uploaded knowledgebase source file.
type KBDocument struct {
	ID           string    `json:"id"`
	DeviceID     string    `json:"device_id"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mime_type"`
	FileSize     int64     `json:"file_size"`
	ChunkCount   int       `json:"chunk_count"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// KBChunk is one retrievable text fragment of a KBDocument.
type KBChunk struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"document_id"`
	DeviceID   string    `json:"device_id"`
	ChunkIndex int       `json:"chunk_index"`
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"`
	CreatedAt  time.Time `json:"created_at"`
}

// RetrievedChunk is a chunk returned by vector search with its score.
type RetrievedChunk struct {
	Chunk    KBChunk `json:"chunk"`
	Distance float64 `json:"distance"` // cosine distance (0 = identical, 2 = opposite)
	Score    float64 `json:"score"`    // 1 - distance/2 mapped to [0,1]
}

// UploadRequest is the wire shape for POST /aireply/documents.
type UploadRequest struct {
	File *multipart.FileHeader `json:"-"`
}

// ChatSetting holds the per-chat AI auto-reply toggle.
type ChatSetting struct {
	DeviceID  string    `json:"device_id"`
	ChatJID   string    `json:"chat_jid"`
	Enabled   bool      `json:"enabled"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatSettingRequest is the wire shape for PUT /aireply/chat-settings/:chat_jid.
type ChatSettingRequest struct {
	Enabled bool `json:"enabled"`
}

// ReplyLog is one audit record of an AI reply attempt.
type ReplyLog struct {
	ID                int64     `json:"id"`
	DeviceID          string    `json:"device_id"`
	ChatJID           string    `json:"chat_jid"`
	Query             string    `json:"query"`
	RetrievedChunkIDs string    `json:"retrieved_chunk_ids"` // JSON array
	Response          string    `json:"response"`
	LatencyMs         int       `json:"latency_ms"`
	TokensIn          int       `json:"tokens_in"`
	TokensOut         int       `json:"tokens_out"`
	Status            string    `json:"status"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// LogFilter filters audit log queries.
type LogFilter struct {
	DeviceID string
	ChatJID  string
	Status   string
	Limit    int
}

// ChatMessage is a provider-agnostic turn in an LLM chat completion.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// ChatRequest is a provider-agnostic chat completion request.
type ChatRequest struct {
	Messages    []ChatMessage
	Model       string
	MaxTokens   int
	Temperature float64
}

// ChatResponse is a provider-agnostic chat completion response.
type ChatResponse struct {
	Content   string
	TokensIn  int
	TokensOut int
}

// IAIProvider abstracts an LLM provider for chat completion and embeddings.
type IAIProvider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	// Embed returns one embedding vector per input text. Float32 to match
	// sqlite-vec storage format.
	Embed(ctx context.Context, model string, inputs []string) ([][]float32, error)
	// Name returns the provider identifier (anthropic|openai_compatible).
	Name() string
	// SupportsEmbeddings indicates whether this provider can produce
	// embeddings natively (Anthropic does not).
	SupportsEmbeddings() bool
}

// IConfigRepository persists per-device AI configuration.
type IConfigRepository interface {
	Get(ctx context.Context, deviceID string) (*AIConfig, error)
	Upsert(ctx context.Context, cfg *AIConfig) error
	Delete(ctx context.Context, deviceID string) error
}

// IKBRepository persists knowledgebase documents and chunks.
type IKBRepository interface {
	CreateDocument(ctx context.Context, doc *KBDocument) error
	UpdateDocumentStatus(ctx context.Context, id, status, errMsg string, chunkCount int) error
	GetDocument(ctx context.Context, deviceID, id string) (*KBDocument, error)
	ListDocuments(ctx context.Context, deviceID string) ([]KBDocument, error)
	DeleteDocument(ctx context.Context, deviceID, id string) error
	InsertChunks(ctx context.Context, chunks []KBChunk) ([]int64, error)
	GetChunksByDocument(ctx context.Context, deviceID, documentID string) ([]KBChunk, error)
	GetChunksByIDs(ctx context.Context, deviceID string, ids []int64) ([]KBChunk, error)
	ListAllChunks(ctx context.Context, deviceID string) ([]KBChunk, error)
}

// IVecStore wraps the sqlite-vec virtual table for vector search.
type IVecStore interface {
	Init(dimension int) error
	Insert(ctx context.Context, deviceID string, chunkID int64, embedding []float32) error
	DeleteByChunkIDs(ctx context.Context, ids []int64) error
	DeleteByDevice(ctx context.Context, deviceID string) error
	Search(ctx context.Context, deviceID string, queryVec []float32, topK int) ([]RetrievedChunk, error)
	Available() bool
}

// IChatSettingRepository persists per-chat toggles.
type IChatSettingRepository interface {
	Get(ctx context.Context, deviceID, chatJID string) (*ChatSetting, error)
	List(ctx context.Context, deviceID string) ([]ChatSetting, error)
	Upsert(ctx context.Context, s *ChatSetting) error
	Delete(ctx context.Context, deviceID, chatJID string) error
}

// ILogRepository persists audit logs.
type ILogRepository interface {
	Insert(ctx context.Context, log *ReplyLog) error
	List(ctx context.Context, filter LogFilter) ([]ReplyLog, error)
}

// IService is the top-level orchestrator + REST-facing usecase.
type IService interface {
	// Inbound message handler. Returns true if AI took ownership of this
	// message (a reply was sent OR a rate limit / out-of-scope decision
	// was made), so the caller can skip the static auto-reply.
	HandleIncoming(ctx context.Context, deviceID, chatJID, senderJID, text string) bool

	// Config CRUD
	GetConfig(ctx context.Context, deviceID string) (*AIConfig, error)
	SaveConfig(ctx context.Context, deviceID string, req AIConfigRequest) error
	TestConfig(ctx context.Context, deviceID string) (latencyMs int, sample string, err error)

	// KB management
	UploadDocument(ctx context.Context, deviceID, filename, mime string, data []byte) (*KBDocument, error)
	ListDocuments(ctx context.Context, deviceID string) ([]KBDocument, error)
	DeleteDocument(ctx context.Context, deviceID, id string) error
	ReindexAll(ctx context.Context, deviceID string) error

	// Chat toggles
	ListChatSettings(ctx context.Context, deviceID string) ([]ChatSetting, error)
	SetChatEnabled(ctx context.Context, deviceID, chatJID string, enabled bool) error

	// Logs
	ListLogs(ctx context.Context, filter LogFilter) ([]ReplyLog, error)
}
