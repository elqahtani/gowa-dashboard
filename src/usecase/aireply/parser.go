package aireply

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	pdfreader "github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
)

// ExtractText extracts plain text from PDF, DOCX, or TXT bytes. The mime
// argument is a best-effort hint; the function falls back to file extension
// detection when mime is empty or generic.
func ExtractText(data []byte, filename, mime string) (string, error) {
	kind := detectKind(filename, mime)
	switch kind {
	case "pdf":
		return extractPDF(data)
	case "docx":
		return extractDOCX(data)
	case "txt":
		return strings.TrimSpace(string(data)), nil
	default:
		return "", fmt.Errorf("unsupported file type: %s (use PDF, DOCX, or TXT)", kind)
	}
}

func detectKind(filename, mime string) string {
	m := strings.ToLower(mime)
	switch {
	case strings.Contains(m, "pdf"):
		return "pdf"
	case strings.Contains(m, "wordprocessingml") || strings.Contains(m, "msword"):
		return "docx"
	case strings.HasPrefix(m, "text/") || strings.Contains(m, "plain"):
		return "txt"
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".txt", ".md":
		return "txt"
	}
	return "unknown"
}

func extractPDF(data []byte) (string, error) {
	reader, err := pdfreader.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	var buf strings.Builder
	totalPages := reader.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n")
	}
	out := strings.TrimSpace(buf.String())
	if len(out) < 50 {
		return "", errors.New("pdf extracted < 50 chars; may be scanned/image-only — convert to TXT manually")
	}
	return out, nil
}

func extractDOCX(data []byte) (string, error) {
	r, err := docx.ReadDocxFromMemory(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()
	content := r.Editable().GetContent()
	// docx.GetContent returns raw XML; strip tags conservatively.
	plain := stripXMLTags(content)
	out := strings.TrimSpace(plain)
	if len(out) < 50 {
		return "", errors.New("docx extracted < 50 chars; document may be empty or unsupported")
	}
	return out, nil
}

// stripXMLTags is a minimal tag remover for DOCX content. It is intentionally
// simple — well-formed DOCX XML has paragraphs in <w:p> and runs in <w:r>; we
// just drop the markup and collapse whitespace.
func stripXMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteRune(r)
		}
	}
	out := b.String()
	// Collapse runs of whitespace and decode common XML entities.
	out = strings.ReplaceAll(out, "&amp;", "&")
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", "\"")
	out = strings.ReplaceAll(out, "&#39;", "'")
	// Normalise newlines.
	fields := strings.Fields(out)
	return strings.Join(fields, " ")
}
