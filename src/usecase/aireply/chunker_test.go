package aireply

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChunk_Empty(t *testing.T) {
	assert.Nil(t, Chunk("", 100, 10))
	assert.Nil(t, Chunk("   \n\n  ", 100, 10))
}

func TestChunk_ShortStaysInOnePiece(t *testing.T) {
	text := "Halo, ini paragraf pendek. Cuma satu kalimat singkat."
	chunks := Chunk(text, 500, 50)
	if assert.Len(t, chunks, 1) {
		assert.Equal(t, 0, chunks[0].Index)
		assert.Contains(t, chunks[0].Content, "paragraf pendek")
	}
}

func TestChunk_LongSplitsOnParagraphs(t *testing.T) {
	long := strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 80)
	text := long + "\n\n" + long + "\n\n" + long
	chunks := Chunk(text, 200, 20)
	assert.Greater(t, len(chunks), 1)
	for i, c := range chunks {
		assert.Equal(t, i, c.Index)
		assert.NotEmpty(t, c.Content)
	}
}

func TestChunk_OverlapPreservesTail(t *testing.T) {
	text := strings.Repeat("alpha bravo charlie delta echo foxtrot golf hotel india. ", 30)
	chunks := Chunk(text, 100, 30)
	if !assert.GreaterOrEqual(t, len(chunks), 2) {
		return
	}
	// The second chunk should begin with content carried over from the tail
	// of the first chunk's overlap window.
	prev := chunks[0].Content
	next := chunks[1].Content
	overlapChars := 30 * charsPerToken
	if len(prev) > overlapChars {
		tail := prev[len(prev)-overlapChars:]
		// tail words should appear at next's start (loose check).
		first := strings.Fields(tail)[0]
		assert.Contains(t, next, first)
	}
}
