package aireply

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
)

// OpenAICompatibleProvider talks to any service that implements the OpenAI
// REST surface (/v1/chat/completions + /v1/embeddings). This covers OpenAI,
// OpenRouter, Sumopod, DeepSeek, Groq, and Ollama (with `openai-compat`
// mode).
type OpenAICompatibleProvider struct {
	client *openai.Client
}

// NewOpenAICompatibleProvider wires a client with an optional custom BaseURL.
// Empty baseURL falls back to OpenAI's official endpoint.
func NewOpenAICompatibleProvider(apiKey, baseURL string) *OpenAICompatibleProvider {
	cfg := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		cfg.BaseURL = strings.TrimRight(baseURL, "/")
	}
	return &OpenAICompatibleProvider{client: openai.NewClientWithConfig(cfg)}
}

func (p *OpenAICompatibleProvider) Name() string         { return domain.ProviderOpenAICompatible }
func (p *OpenAICompatibleProvider) SupportsEmbeddings() bool { return true }

func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	timeout := time.Duration(config.AIRequestTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := p.client.CreateChatCompletion(cctx, openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
	})
	if err != nil {
		return domain.ChatResponse{}, err
	}
	if len(resp.Choices) == 0 {
		return domain.ChatResponse{}, errors.New("openai: empty choices")
	}
	return domain.ChatResponse{
		Content:   strings.TrimSpace(resp.Choices[0].Message.Content),
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (p *OpenAICompatibleProvider) Embed(ctx context.Context, model string, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	timeout := time.Duration(config.AIRequestTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := p.client.CreateEmbeddings(cctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(model),
		Input: inputs,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	if len(resp.Data) != len(inputs) {
		return nil, fmt.Errorf("openai embed: got %d vectors for %d inputs", len(resp.Data), len(inputs))
	}
	out := make([][]float32, len(resp.Data))
	for i, d := range resp.Data {
		out[i] = d.Embedding
	}
	return out, nil
}
