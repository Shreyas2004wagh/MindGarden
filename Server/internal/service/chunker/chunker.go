package chunker

import (
	"strings"
	"unicode"
)

// ChunkConfig defines chunking parameters
type ChunkConfig struct {
	MaxChunkSize int // Maximum characters per chunk
	MinChunkSize int // Minimum characters per chunk
	Overlap      int // Character overlap between chunks
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	Content    string
	Index      int
	TotalCount int
}

// DefaultConfig returns sensible defaults for journal chunking
func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize: 600, // ~90-120 words (2-3 paragraphs)
		MinChunkSize: 300, // ~45-60 words (1 paragraph)
		Overlap:      80,  // ~12-15 words overlap for context
	}
}

// ChunkText splits text into semantic chunks with overlap
func ChunkText(text string, config ChunkConfig) []Chunk {
	text = strings.TrimSpace(text)

	// If text is short, return as single chunk
	if len(text) <= config.MaxChunkSize {
		return []Chunk{
			{
				Content:    text,
				Index:      0,
				TotalCount: 1,
			},
		}
	}

	// Split into paragraphs first (semantic boundaries)
	paragraphs := splitIntoParagraphs(text)

	var chunks []Chunk
	var currentChunk strings.Builder
	var chunkIndex int

	for i, para := range paragraphs {
		paraLen := len(para)
		currentLen := currentChunk.Len()

		// If this single paragraph is longer than max chunk size, split it into sentences
		if paraLen > config.MaxChunkSize {
			// Save current chunk if it has content
			if currentLen > 0 {
				chunks = append(chunks, Chunk{
					Content:    currentChunk.String(),
					Index:      chunkIndex,
					TotalCount: 0,
				})
				chunkIndex++
				currentChunk.Reset()
			}

			// Split long paragraph into sentences and chunk them
			sentences := splitIntoSentences(para)
			for _, sentence := range sentences {
				sentenceLen := len(sentence)
				currentLen = currentChunk.Len()

				if currentLen > 0 && currentLen+sentenceLen+1 > config.MaxChunkSize {
					// Save current chunk
					chunks = append(chunks, Chunk{
						Content:    currentChunk.String(),
						Index:      chunkIndex,
						TotalCount: 0,
					})
					chunkIndex++

					// Start new chunk with overlap
					currentChunk.Reset()
					if len(chunks) > 0 {
						overlap := getOverlap(chunks[len(chunks)-1].Content, config.Overlap)
						if overlap != "" {
							currentChunk.WriteString(overlap)
							currentChunk.WriteString(" ")
						}
					}
				}

				// Add sentence to current chunk
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(sentence)
			}
			continue
		}

		// If adding this paragraph exceeds max size
		if currentLen > 0 && currentLen+paraLen+1 > config.MaxChunkSize {
			// Save current chunk
			chunks = append(chunks, Chunk{
				Content:    currentChunk.String(),
				Index:      chunkIndex,
				TotalCount: 0, // Will be set later
			})
			chunkIndex++

			// Start new chunk with overlap from previous
			currentChunk.Reset()
			if len(chunks) > 0 {
				overlap := getOverlap(chunks[len(chunks)-1].Content, config.Overlap)
				if overlap != "" {
					currentChunk.WriteString(overlap)
					currentChunk.WriteString("\n\n")
				}
			}
		}

		// Add paragraph to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)

		// If this is the last paragraph, save the chunk
		if i == len(paragraphs)-1 {
			chunks = append(chunks, Chunk{
				Content:    currentChunk.String(),
				Index:      chunkIndex,
				TotalCount: 0,
			})
		}
	}

	// Set total count for all chunks
	totalCount := len(chunks)
	for i := range chunks {
		chunks[i].TotalCount = totalCount
	}

	return chunks
}

// splitIntoParagraphs splits text by double newlines or single newlines
func splitIntoParagraphs(text string) []string {
	// First try splitting by double newlines (clear paragraph breaks)
	paragraphs := strings.Split(text, "\n\n")

	var result []string
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If a "paragraph" is too long, split by single newlines
		if len(para) > 1500 {
			lines := strings.Split(para, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					result = append(result, line)
				}
			}
		} else {
			result = append(result, para)
		}
	}

	return result
}

// splitIntoSentences splits text into sentences based on punctuation
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		// Check for sentence endings: . ! ?
		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// Look ahead to see if there's a space (end of sentence)
			if i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add remaining text
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	// If no sentences were found, return the whole text
	if len(sentences) == 0 {
		sentences = append(sentences, strings.TrimSpace(text))
	}

	return sentences
}

// getOverlap returns the last N characters from text, preferring sentence boundaries
func getOverlap(text string, overlapSize int) string {
	if len(text) <= overlapSize {
		return text
	}

	// Get the last overlapSize characters
	start := len(text) - overlapSize
	overlap := text[start:]

	// Try to start at a sentence boundary (. ! ?)
	sentenceEnders := []rune{'.', '!', '?'}
	for i, r := range overlap {
		for _, ender := range sentenceEnders {
			if r == ender && i < len(overlap)-1 && unicode.IsSpace(rune(overlap[i+1])) {
				return strings.TrimSpace(overlap[i+1:])
			}
		}
	}

	// If no sentence boundary, try word boundary
	for i, r := range overlap {
		if unicode.IsSpace(r) {
			return strings.TrimSpace(overlap[i:])
		}
	}

	// Fallback: return as-is
	return overlap
}
