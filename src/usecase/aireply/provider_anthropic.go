package aireply

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
)

// AnthropicProvider calls Anthropic's /v1/messages endpoint directly via HTTP
// so we don't pull in the official SDK's transitive deps. Anthropic does not
// expose embeddings — callers should configure a separate OpenAI-compatible
// embed provider.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

const defaultAnthropicBaseURL = "https://api.anthropic.com"
const anthropicVersion = "2023-06-01"

func NewAnthropicProvider(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
	}
}

func (p *AnthropicProvider) Name() string             { return domain.ProviderAnthropic }
func (p *AnthropicProvider) SupportsEmbeddings() bool { return false }

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicReq struct {
	Model       string         `json:"model"`
	System      string         `json:"system,omitempty"`
	Messages    []anthropicMsg `json:"messages"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResp struct {
	Content []anthropicContentBlock `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	if p.apiKey == "" {
		return domain.ChatResponse{}, errors.New("anthropic: api key not set")
	}

	// Anthropic separates system prompt from messages array.
	var system string
	msgs := make([]anthropicMsg, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		msgs = append(msgs, anthropicMsg{Role: m.Role, Content: m.Content})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 500
	}
	payload := anthropicReq{
		Model:       req.Model,
		System:      system,
		Messages:    msgs,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.ChatResponse{}, err
	}

	timeout := time.Duration(config.AIRequestTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(cctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return domain.ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return domain.ChatResponse{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var parsed anthropicResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return domain.ChatResponse{}, fmt.Errorf("anthropic: decode response (status %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode >= 400 {
		if parsed.Error != nil {
			return domain.ChatResponse{}, fmt.Errorf("anthropic %s: %s", parsed.Error.Type, parsed.Error.Message)
		}
		return domain.ChatResponse{}, fmt.Errorf("anthropic: http %d", resp.StatusCode)
	}

	var text strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			text.WriteString(c.Text)
		}
	}
	return domain.ChatResponse{
		Content:   strings.TrimSpace(text.String()),
		TokensIn:  parsed.Usage.InputTokens,
		TokensOut: parsed.Usage.OutputTokens,
	}, nil
}

// Embed is intentionally not supported; the orchestrator must route embeddings
// to a separately-configured OpenAI-compatible provider.
func (p *AnthropicProvider) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	return nil, errors.New("anthropic: embeddings not supported; configure embed_provider=openai_compatible")
}
