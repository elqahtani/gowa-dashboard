package aireply

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
)

func TestBuildPrompt_GuardrailIncluded(t *testing.T) {
	cfg := &domain.AIConfig{
		StylePreset:       domain.StyleCustomerServiceFormal,
		GuardrailEnabled:  true,
		OutOfScopeMessage: "Maaf, di luar topik kami.",
	}
	retrieved := []RetrievedChunkContent{{Score: 0.9, Content: "Harga Produk A adalah Rp100.000."}}
	msgs := BuildPrompt(cfg, "Berapa harga Produk A?", retrieved, nil)
	if !assert.GreaterOrEqual(t, len(msgs), 2) {
		return
	}
	assert.Equal(t, "system", msgs[0].Role)
	assert.Contains(t, msgs[0].Content, "ATURAN PENTING")
	assert.Contains(t, msgs[0].Content, "Maaf, di luar topik kami.")
	assert.Contains(t, msgs[0].Content, "Harga Produk A")
	assert.Equal(t, "user", msgs[len(msgs)-1].Role)
	assert.Equal(t, "Berapa harga Produk A?", msgs[len(msgs)-1].Content)
}

func TestBuildPrompt_CustomStyleUsesSystemPrompt(t *testing.T) {
	cfg := &domain.AIConfig{
		StylePreset:  domain.StyleCustom,
		SystemPrompt: "Kamu adalah bot eksperimental yang hanya jawab dengan satu kata.",
	}
	msgs := BuildPrompt(cfg, "halo", nil, nil)
	assert.Contains(t, msgs[0].Content, "bot eksperimental")
}

func TestBuildPrompt_HistoryRolesFlippedCorrectly(t *testing.T) {
	cfg := &domain.AIConfig{StylePreset: domain.StyleSalesCasual}
	hist := []HistoryMessage{
		{IsFromMe: false, Content: "Hi"},
		{IsFromMe: true, Content: "Hai! Ada yang bisa dibantu?"},
	}
	msgs := BuildPrompt(cfg, "Mau tanya produk", nil, hist)
	// system + 2 history + 1 user
	if !assert.Len(t, msgs, 4) {
		return
	}
	assert.Equal(t, "user", msgs[1].Role)
	assert.Equal(t, "assistant", msgs[2].Role)
	assert.Equal(t, "user", msgs[3].Role)
	assert.True(t, strings.HasPrefix(msgs[2].Content, "Hai"))
}
