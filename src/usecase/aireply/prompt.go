package aireply

import (
	"fmt"
	"strings"

	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
)

// StylePresets maps style id → system prompt template (Bahasa Indonesia). Used
// when AIConfig.SystemPrompt is empty for non-custom presets.
var StylePresets = map[string]string{
	domain.StyleCustomerServiceFormal: "Anda adalah customer service profesional. Jawab dengan sopan, jelas, dan rapi menggunakan bahasa Indonesia formal. Sapa singkat di awal (mis. 'Halo,') dan tutup dengan kalimat yang menanyakan apakah ada yang bisa dibantu lagi.",
	domain.StyleSalesCasual:           "Kamu adalah sales yang ramah dan helpful. Gunakan bahasa Indonesia santai dengan sapaan kasual (mis. 'Hai!'). Fokus highlight value dan benefit produk dari informasi yang tersedia di knowledgebase.",
	domain.StyleTechnicalSupport:      "Anda adalah technical support. Jawab to-the-point, sertakan langkah-langkah berurutan (numbered list) kalau perlu troubleshooting. Bahasa Indonesia campur istilah teknis Inggris diperbolehkan.",
	domain.StyleCustom:                "",
}

// BuildPrompt assembles the full chat messages array for the LLM call.
//
// Layout:
//   system: style + guardrail + <context> + (optional explicit user override)
//   ...history (last N messages, oldest first)
//   user:   query
func BuildPrompt(cfg *domain.AIConfig, query string, retrieved []RetrievedChunkContent, history []HistoryMessage) []domain.ChatMessage {
	var sys strings.Builder

	stylePart := cfg.SystemPrompt
	if strings.TrimSpace(stylePart) == "" {
		stylePart = StylePresets[cfg.StylePreset]
	}
	if stylePart != "" {
		sys.WriteString(stylePart)
		sys.WriteString("\n\n")
	}

	if cfg.GuardrailEnabled {
		oos := cfg.OutOfScopeMessage
		if strings.TrimSpace(oos) == "" {
			oos = domain.DefaultOutOfScopeMessage
		}
		sys.WriteString("ATURAN PENTING (WAJIB DIIKUTI):\n")
		sys.WriteString("- Jawab HANYA berdasarkan informasi di dalam tag <context> dan riwayat chat di bawah.\n")
		sys.WriteString("- Jika informasi yang ditanyakan TIDAK ADA di context, balas PERSIS dengan kalimat berikut, tanpa tambahan apa pun:\n")
		sys.WriteString("  \"" + oos + "\"\n")
		sys.WriteString("- DILARANG mengarang fakta atau memakai pengetahuan umum di luar context.\n")
		sys.WriteString("- DILARANG menyebutkan bahwa kamu adalah AI atau menyebut tentang context/knowledgebase secara eksplisit ke user.\n")
		sys.WriteString("- Jaga panjang balasan singkat dan natural seperti chat WhatsApp.\n\n")
	}

	sys.WriteString("<context>\n")
	if len(retrieved) == 0 {
		sys.WriteString("(no relevant knowledge)\n")
	} else {
		for i, r := range retrieved {
			fmt.Fprintf(&sys, "[chunk %d | score=%.2f]\n%s\n\n", i+1, r.Score, r.Content)
		}
	}
	sys.WriteString("</context>")

	msgs := make([]domain.ChatMessage, 0, len(history)+2)
	msgs = append(msgs, domain.ChatMessage{Role: "system", Content: sys.String()})
	for _, h := range history {
		role := "user"
		if h.IsFromMe {
			role = "assistant"
		}
		if strings.TrimSpace(h.Content) == "" {
			continue
		}
		msgs = append(msgs, domain.ChatMessage{Role: role, Content: h.Content})
	}
	msgs = append(msgs, domain.ChatMessage{Role: "user", Content: query})
	return msgs
}

// HistoryMessage is a minimal chat message representation for prompt history.
type HistoryMessage struct {
	IsFromMe bool
	Content  string
}

// RetrievedChunkContent pairs the retrieved score with the chunk's text body.
type RetrievedChunkContent struct {
	Score   float64
	Content string
}
