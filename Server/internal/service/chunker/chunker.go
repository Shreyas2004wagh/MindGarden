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

// EnrichedChunk extends Chunk with semantic metadata
type EnrichedChunk struct {
	Chunk
	WordCount     int      // Number of words in the chunk
	SentenceCount int      // Number of sentences in the chunk
	HasQuestions  bool     // Whether the chunk contains questions
	HasDates      bool     // Whether the chunk contains date references
	Sentiment     string   // Sentiment: positive, negative, neutral
	Topics        []string // Extracted topics/keywords
}

// DefaultConfig returns sensible defaults for journal chunking
func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize: 600, // ~90-120 words (2-3 paragraphs)
		MinChunkSize: 300, // ~45-60 words (1 paragraph)
		Overlap:      80,  // ~12-15 words overlap for context
	}
}

// GetOptimalConfig returns adaptive chunking configuration based on text length
func GetOptimalConfig(textLength int) ChunkConfig {
	switch {
	case textLength < 500:
		// Short journal - don't chunk
		return ChunkConfig{
			MaxChunkSize: 1000,
			MinChunkSize: 500,
			Overlap:      0,
		}
	case textLength < 1500:
		// Medium journal - small chunks
		return ChunkConfig{
			MaxChunkSize: 600,
			MinChunkSize: 300,
			Overlap:      80,
		}
	case textLength < 3000:
		// Long journal - medium chunks
		return ChunkConfig{
			MaxChunkSize: 800,
			MinChunkSize: 400,
			Overlap:      100,
		}
	default:
		// Very long journal - larger chunks
		return ChunkConfig{
			MaxChunkSize: 1000,
			MinChunkSize: 500,
			Overlap:      150,
		}
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

// ChunkTextEnriched splits text into enriched chunks with metadata
func ChunkTextEnriched(text string, config ChunkConfig) []EnrichedChunk {
	baseChunks := ChunkText(text, config)
	enrichedChunks := make([]EnrichedChunk, len(baseChunks))

	for i, chunk := range baseChunks {
		enrichedChunks[i] = EnrichedChunk{
			Chunk:         chunk,
			WordCount:     countWords(chunk.Content),
			SentenceCount: countSentences(chunk.Content),
			HasQuestions:  hasQuestions(chunk.Content),
			HasDates:      hasDates(chunk.Content),
			Sentiment:     analyzeSentiment(chunk.Content),
			Topics:        extractTopics(chunk.Content),
		}
	}

	return enrichedChunks
}

// countWords counts the number of words in text
func countWords(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return len(words)
}

// countSentences counts the number of sentences in text
func countSentences(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	sentences := splitIntoSentences(text)
	return len(sentences)
}

// hasQuestions checks if text contains question marks
func hasQuestions(text string) bool {
	return strings.Contains(text, "?")
}

// hasDates checks if text contains date patterns
func hasDates(text string) bool {
	// Common date patterns: "January", "2024", "01/01", "1st", "today", "yesterday", etc.
	dateKeywords := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
		"jan", "feb", "mar", "apr", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
		"today", "yesterday", "tomorrow", "monday", "tuesday", "wednesday",
		"thursday", "friday", "saturday", "sunday",
	}

	lowerText := strings.ToLower(text)
	for _, keyword := range dateKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}

	// Check for numeric date patterns (e.g., "2024", "01/01", "1st", "2nd")
	for _, r := range text {
		if unicode.IsDigit(r) {
			// If we find digits, check for common date patterns
			if strings.Contains(text, "/") || strings.Contains(text, "-") ||
				strings.Contains(text, "st") || strings.Contains(text, "nd") ||
				strings.Contains(text, "rd") || strings.Contains(text, "th") {
				return true
			}
		}
	}

	return false
}

// analyzeSentiment performs simple keyword-based sentiment analysis
func analyzeSentiment(text string) string {
	lowerText := strings.ToLower(text)

	positiveWords := []string{
		"happy", "joy", "love", "excited", "great", "wonderful", "amazing",
		"fantastic", "excellent", "good", "better", "best", "grateful",
		"thankful", "blessed", "proud", "accomplished", "success", "win",
	}

	negativeWords := []string{
		"sad", "angry", "hate", "terrible", "awful", "bad", "worse", "worst",
		"depressed", "anxious", "worried", "stressed", "frustrated", "upset",
		"disappointed", "fail", "failure", "lost", "hurt", "pain",
	}

	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		positiveCount += strings.Count(lowerText, word)
	}

	for _, word := range negativeWords {
		negativeCount += strings.Count(lowerText, word)
	}

	// Determine sentiment based on counts
	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	return "neutral"
}

// extractTopics extracts key topics using simple word frequency analysis
func extractTopics(text string) []string {
	// Common stop words to ignore
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "been": true, "be": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "may": true, "might": true, "must": true,
		"i": true, "you": true, "he": true, "she": true, "it": true, "we": true,
		"they": true, "my": true, "your": true, "his": true, "her": true, "its": true,
		"our": true, "their": true, "this": true, "that": true, "these": true, "those": true,
	}

	// Clean and split text into words
	lowerText := strings.ToLower(text)
	// Remove punctuation
	cleanText := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, lowerText)

	words := strings.Fields(cleanText)
	wordFreq := make(map[string]int)

	// Count word frequencies
	for _, word := range words {
		if len(word) > 3 && !stopWords[word] { // Only consider words longer than 3 chars
			wordFreq[word]++
		}
	}

	// Extract top topics (words with frequency > 1)
	var topics []string
	for word, freq := range wordFreq {
		if freq > 1 {
			topics = append(topics, word)
		}
	}

	// Limit to top 5 topics
	if len(topics) > 5 {
		topics = topics[:5]
	}

	return topics
}

// endsWithSentence checks if text ends with a sentence boundary
func endsWithSentence(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}
	lastChar := rune(text[len(text)-1])
	return lastChar == '.' || lastChar == '!' || lastChar == '?'
}
