package validations

import (
	"errors"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
	pkgError "github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/error"
)

// ValidateAIConfigRequest checks an incoming PUT /aireply/config body.
func ValidateAIConfigRequest(req domain.AIConfigRequest) error {
	if err := validation.ValidateStruct(&req,
		validation.Field(&req.Provider, validation.Required, validation.In(domain.ProviderAnthropic, domain.ProviderOpenAICompatible)),
		validation.Field(&req.Model, validation.Required, validation.Length(1, 255)),
		validation.Field(&req.StylePreset, validation.Required, validation.In(
			domain.StyleCustomerServiceFormal, domain.StyleSalesCasual,
			domain.StyleTechnicalSupport, domain.StyleCustom,
		)),
		validation.Field(&req.BaseURL, validation.When(req.BaseURL != "", is.URL)),
		validation.Field(&req.EmbedBaseURL, validation.When(req.EmbedBaseURL != "", is.URL)),
		validation.Field(&req.MaxTokens, validation.Min(1), validation.Max(8000)),
		validation.Field(&req.Temperature, validation.Min(0.0), validation.Max(2.0)),
		validation.Field(&req.TopK, validation.Min(1), validation.Max(20)),
		validation.Field(&req.RetrievalThreshold, validation.Min(0.0), validation.Max(1.0)),
	); err != nil {
		return pkgError.ValidationError(err.Error())
	}
	if req.Provider == domain.ProviderAnthropic && req.EmbedProvider == "" {
		return pkgError.ValidationError("anthropic chat requires embed_provider (e.g. openai_compatible) with embed_model")
	}
	if req.StylePreset == domain.StyleCustom && strings.TrimSpace(req.SystemPrompt) == "" {
		return pkgError.ValidationError("style_preset=custom requires a non-empty system_prompt")
	}
	if config.AIEncryptionKey == "" {
		return pkgError.ValidationError("AI_ENCRYPTION_KEY env var is not set — required to encrypt provider api keys")
	}
	return nil
}

// ValidateKBUpload checks size + filename hint.
func ValidateKBUpload(filename string, size int64) error {
	if strings.TrimSpace(filename) == "" {
		return pkgError.ValidationError("filename required")
	}
	if size <= 0 {
		return pkgError.ValidationError("empty file")
	}
	if size > config.AIMaxKBFileSize {
		return pkgError.ValidationError("file exceeds AI_MAX_KB_FILE_SIZE")
	}
	lower := strings.ToLower(filename)
	if !strings.HasSuffix(lower, ".pdf") &&
		!strings.HasSuffix(lower, ".docx") &&
		!strings.HasSuffix(lower, ".txt") &&
		!strings.HasSuffix(lower, ".md") {
		return pkgError.ValidationError("unsupported format: only .pdf, .docx, .txt, .md")
	}
	return nil
}

// ValidateChatJID is a small wrapper used by toggle endpoints.
func ValidateChatJID(jid string) error {
	if strings.TrimSpace(jid) == "" {
		return pkgError.ValidationError("chat_jid required")
	}
	if !strings.Contains(jid, "@") {
		return errors.New("invalid chat_jid (missing server)")
	}
	return nil
}
