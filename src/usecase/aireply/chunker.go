package aireply

import "strings"

// TextChunk is one piece of text + an approximate token count.
type TextChunk struct {
	Index      int
	Content    string
	TokenCount int
}

// charsPerToken is a rough approximation for English/Indonesian mixed text.
// Good enough for chunk-size budgeting; not used for billing.
const charsPerToken = 4

// Chunk splits text into chunks of ~maxTokens tokens with overlap. Splits
// prefer paragraph boundaries, falling back to sentence then word boundaries.
//
// Defaults: maxTokens=500, overlap=50 if either is non-positive.
func Chunk(text string, maxTokens, overlap int) []TextChunk {
	if maxTokens <= 0 {
		maxTokens = 500
	}
	if overlap < 0 || overlap >= maxTokens {
		overlap = 50
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	maxChars := maxTokens * charsPerToken
	overlapChars := overlap * charsPerToken

	// Split into paragraphs first to keep semantic boundaries.
	paras := splitParagraphs(text)
	var chunks []TextChunk
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s == "" {
			return
		}
		chunks = append(chunks, TextChunk{
			Index:      len(chunks),
			Content:    s,
			TokenCount: len(s) / charsPerToken,
		})
		// Carry tail as overlap into next buffer.
		if overlapChars > 0 && len(s) > overlapChars {
			tail := s[len(s)-overlapChars:]
			buf.Reset()
			buf.WriteString(tail)
			buf.WriteString("\n\n")
		} else {
			buf.Reset()
		}
	}

	for _, p := range paras {
		if buf.Len()+len(p)+2 <= maxChars {
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(p)
			continue
		}
		// Paragraph would overflow — flush current then handle it.
		flush()
		if len(p) <= maxChars {
			buf.WriteString(p)
			continue
		}
		// Paragraph too big on its own — split by sentence then word.
		for _, sub := range splitLongUnit(p, maxChars) {
			if buf.Len()+len(sub)+1 > maxChars {
				flush()
			}
			if buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteString(sub)
		}
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	// Normalise CRLF and split on blank lines.
	t := strings.ReplaceAll(text, "\r\n", "\n")
	parts := strings.Split(t, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 && strings.TrimSpace(t) != "" {
		out = append(out, strings.TrimSpace(t))
	}
	return out
}

// splitLongUnit breaks a too-large paragraph into max-sized sub-units,
// preferring sentence then word boundaries.
func splitLongUnit(p string, maxChars int) []string {
	sentences := splitSentences(p)
	var out []string
	var buf strings.Builder
	for _, s := range sentences {
		if len(s) > maxChars {
			// Sentence itself too long — split by words.
			if buf.Len() > 0 {
				out = append(out, strings.TrimSpace(buf.String()))
				buf.Reset()
			}
			out = append(out, splitByWords(s, maxChars)...)
			continue
		}
		if buf.Len()+len(s)+1 > maxChars {
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(s)
	}
	if buf.Len() > 0 {
		out = append(out, strings.TrimSpace(buf.String()))
	}
	return out
}

func splitSentences(p string) []string {
	// Simple heuristic split on ".", "!", "?" followed by space/newline.
	var out []string
	var buf strings.Builder
	runes := []rune(p)
	for i, r := range runes {
		buf.WriteRune(r)
		if (r == '.' || r == '!' || r == '?') && (i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n') {
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		out = append(out, strings.TrimSpace(buf.String()))
	}
	return out
}

func splitByWords(s string, maxChars int) []string {
	words := strings.Fields(s)
	var out []string
	var buf strings.Builder
	for _, w := range words {
		if buf.Len()+len(w)+1 > maxChars {
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(w)
	}
	if buf.Len() > 0 {
		out = append(out, strings.TrimSpace(buf.String()))
	}
	return out
}
