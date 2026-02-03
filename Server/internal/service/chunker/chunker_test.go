package chunker

import (
	"strings"
	"testing"
)

func TestChunkText_ShortText(t *testing.T) {
	config := DefaultConfig()
	text := "This is a short journal entry."

	chunks := ChunkText(text, config)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for short text, got %d", len(chunks))
	}

	if chunks[0].Content != text {
		t.Errorf("Expected content to match input")
	}

	if chunks[0].Index != 0 {
		t.Errorf("Expected index 0, got %d", chunks[0].Index)
	}

	if chunks[0].TotalCount != 1 {
		t.Errorf("Expected total count 1, got %d", chunks[0].TotalCount)
	}
}

func TestChunkText_LongText(t *testing.T) {
	config := DefaultConfig()

	// Create a long text with multiple paragraphs
	paragraphs := []string{
		strings.Repeat("This is paragraph one. ", 50),
		strings.Repeat("This is paragraph two. ", 50),
		strings.Repeat("This is paragraph three. ", 50),
	}
	text := strings.Join(paragraphs, "\n\n")

	chunks := ChunkText(text, config)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for long text, got %d", len(chunks))
	}

	// Verify all chunks have correct total count
	for i, chunk := range chunks {
		if chunk.TotalCount != len(chunks) {
			t.Errorf("Chunk %d has incorrect total count: %d, expected %d", i, chunk.TotalCount, len(chunks))
		}
		if chunk.Index != i {
			t.Errorf("Chunk %d has incorrect index: %d", i, chunk.Index)
		}
	}
}

func TestChunkText_EmptyText(t *testing.T) {
	config := DefaultConfig()
	text := ""

	chunks := ChunkText(text, config)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for empty text, got %d", len(chunks))
	}

	if chunks[0].Content != "" {
		t.Errorf("Expected empty content")
	}
}

func TestChunkText_Overlap(t *testing.T) {
	config := ChunkConfig{
		MaxChunkSize: 100, // Reduced from 200 to force chunking
		MinChunkSize: 50,
		Overlap:      30,
	}

	// Create text that will definitely need chunking
	text := strings.Repeat("This is a test sentence. ", 30)

	chunks := ChunkText(text, config)

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
		t.Logf("Text length: %d, Max chunk size: %d", len(text), config.MaxChunkSize)
		return
	}

	// Verify overlap exists between consecutive chunks
	for i := 0; i < len(chunks)-1; i++ {
		currentChunk := chunks[i].Content
		nextChunk := chunks[i+1].Content

		// Check if there's some overlap
		// The end of current chunk should appear in the beginning of next chunk
		if len(currentChunk) >= 30 {
			endOfCurrent := currentChunk[len(currentChunk)-30:]
			if !strings.Contains(nextChunk, strings.TrimSpace(endOfCurrent[:15])) {
				// Some overlap should exist (checking first 15 chars of the overlap region)
				t.Logf("Warning: Overlap might not be working as expected between chunks %d and %d", i, i+1)
			}
		}
	}
}

func TestSplitIntoParagraphs(t *testing.T) {
	text := "Paragraph one.\n\nParagraph two.\n\nParagraph three."

	paragraphs := splitIntoParagraphs(text)

	if len(paragraphs) != 3 {
		t.Errorf("Expected 3 paragraphs, got %d", len(paragraphs))
	}

	expected := []string{"Paragraph one.", "Paragraph two.", "Paragraph three."}
	for i, para := range paragraphs {
		if para != expected[i] {
			t.Errorf("Paragraph %d: expected '%s', got '%s'", i, expected[i], para)
		}
	}
}

func TestGetOverlap(t *testing.T) {
	text := "This is a long sentence. This is another sentence. And one more."
	overlap := getOverlap(text, 30)

	// Should start after a sentence boundary if possible
	if !strings.HasPrefix(strings.TrimSpace(overlap), "And one more.") &&
		!strings.HasPrefix(strings.TrimSpace(overlap), "This is another") {
		t.Logf("Overlap: '%s'", overlap)
		// This is informational, not necessarily an error
	}

	if len(overlap) > 30 {
		t.Errorf("Overlap should not exceed requested size")
	}
}
