package chunker

import (
	"context"
	"regexp"
	"strings"
	"unicode"
)

// ChunkConfig defines chunking parameters
type ChunkConfig struct {
	MaxChunkSize int // Maximum characters per chunk
	MinChunkSize int // Minimum characters per chunk
	Overlap      int // Character overlap between chunks
}

// Default chunking constants
const (
	// Text length thresholds
	JournalLengthShort  = 500
	JournalLengthMedium = 1500
	JournalLengthLong   = 3000

	// Chunk size configurations
	ChunkSizeShortMax = 1000
	ChunkSizeShortMin = 500
	ChunkOverlapShort = 0

	ChunkSizeMediumMax = 600
	ChunkSizeMediumMin = 300
	ChunkOverlapMedium = 80

	ChunkSizeLongMax = 800
	ChunkSizeLongMin = 400
	ChunkOverlapLong = 100

	ChunkSizeVeryLongMax = 1000
	ChunkSizeVeryLongMin = 500
	ChunkOverlapVeryLong = 150
)

// Chunk represents a text chunk with metadata
type Chunk struct {
	Content    string
	Index      int
	TotalCount int
}

// EnrichedChunk extends Chunk with semantic metadata
type EnrichedChunk struct {
	Chunk
	WordCount        int      // Number of words in the chunk
	SentenceCount    int      // Number of sentences in the chunk
	HasQuestions     bool     // Whether the chunk contains questions
	HasDates         bool     // Whether the chunk contains date references
	Sentiment        string   // Sentiment: positive, negative, neutral
	Topics           []string // Extracted topics/keywords
	EnrichmentMethod string   // Method used: "gemini" or "simple"
}

// DefaultConfig returns sensible defaults for journal chunking
func DefaultConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize: ChunkSizeMediumMax,
		MinChunkSize: ChunkSizeMediumMin,
		Overlap:      ChunkOverlapMedium,
	}
}

// GetOptimalConfig returns adaptive chunking configuration based on text length
func GetOptimalConfig(textLength int) ChunkConfig {
	switch {
	case textLength < JournalLengthShort:
		// Short journal - don't chunk
		return ChunkConfig{
			MaxChunkSize: ChunkSizeShortMax,
			MinChunkSize: ChunkSizeShortMin,
			Overlap:      ChunkOverlapShort,
		}
	case textLength < JournalLengthMedium:
		// Medium journal - small chunks
		return ChunkConfig{
			MaxChunkSize: ChunkSizeMediumMax,
			MinChunkSize: ChunkSizeMediumMin,
			Overlap:      ChunkOverlapMedium,
		}
	case textLength < JournalLengthLong:
		// Long journal - medium chunks
		return ChunkConfig{
			MaxChunkSize: ChunkSizeLongMax,
			MinChunkSize: ChunkSizeLongMin,
			Overlap:      ChunkOverlapLong,
		}
	default:
		// Very long journal - larger chunks
		return ChunkConfig{
			MaxChunkSize: ChunkSizeVeryLongMax,
			MinChunkSize: ChunkSizeVeryLongMin,
			Overlap:      ChunkOverlapVeryLong,
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
			sentences := SplitIntoSentences(para)
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

// SplitIntoSentences splits text into sentences based on punctuation
func SplitIntoSentences(text string) []string {
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
// If llmService is provided, uses Gemini API for sentiment/topic analysis
// Otherwise falls back to simple keyword-based analysis
func ChunkTextEnriched(text string, config ChunkConfig, llmService LLMService) []EnrichedChunk {
	baseChunks := ChunkText(text, config)
	enrichedChunks := make([]EnrichedChunk, len(baseChunks))

	for i, chunk := range baseChunks {
		enrichedChunk := EnrichedChunk{
			Chunk:         chunk,
			WordCount:     countWords(chunk.Content),
			SentenceCount: countSentences(chunk.Content),
			HasQuestions:  hasQuestions(chunk.Content),
			HasDates:      hasDates(chunk.Content),
		}

		// Try Gemini API if service is provided
		if llmService != nil {
			ctx := context.Background()
			analysis, err := llmService.AnalyzeChunkMetadata(ctx, chunk.Content)
			if err == nil && analysis != nil {
				// Successfully got Gemini analysis
				enrichedChunk.Sentiment = analysis.Sentiment
				enrichedChunk.Topics = analysis.Topics
				enrichedChunk.EnrichmentMethod = "gemini"
			} else {
				// Fallback to simple analysis
				// Log the error for debugging
				if err != nil {
					// Error occurred, using fallback
				}
				enrichedChunk.Sentiment = analyzeSentiment(chunk.Content)
				enrichedChunk.Topics = extractTopics(chunk.Content)
				enrichedChunk.EnrichmentMethod = "simple"
			}
		} else {
			// No LLM service, use simple analysis
			enrichedChunk.Sentiment = analyzeSentiment(chunk.Content)
			enrichedChunk.Topics = extractTopics(chunk.Content)
			enrichedChunk.EnrichmentMethod = "simple"
		}

		enrichedChunks[i] = enrichedChunk
	}

	return enrichedChunks
}

// MetadataAnalysis holds sentiment and topic analysis results
type MetadataAnalysis struct {
	Sentiment string
	Topics    []string
}

// LLMService interface for metadata analysis
type LLMService interface {
	AnalyzeChunkMetadata(ctx context.Context, text string) (*MetadataAnalysis, error)
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
	sentences := SplitIntoSentences(text)
	return len(sentences)
}

// hasQuestions checks if text contains question marks
func hasQuestions(text string) bool {
	return strings.Contains(text, "?")
}

// hasDates checks if text contains date patterns using regex
func hasDates(text string) bool {
	// 1. Common Month Names (Jan, January, etc.) + Day
	// Matches: "January 1", "Jan 1st", "10th of May", "May 10"
	monthRegex := regexp.MustCompile(`(?i)\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+\d{1,2}(?:st|nd|rd|th)?\b|\b\d{1,2}(?:st|nd|rd|th)?\s+of\s+(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:tember)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\b`)

	// 2. Numeric Dates
	// Matches: "2024-01-01", "01/01/2024", "1-1-24", "1.1.2024"
	// Be careful with simple 1/2 fractions implies checking minimal length or context,
	// but strictly for dates usually YYYY is 4 digits or DD/MM/YY
	numericRegex := regexp.MustCompile(`\b\d{4}[-/]\d{1,2}[-/]\d{1,2}\b|\b\d{1,2}[-/]\d{1,2}[-/]\d{2,4}\b`)

	// 3. Relative Days
	// Matches: "today", "tomorrow", "yesterday", "Monday", etc.
	relativeRegex := regexp.MustCompile(`(?i)\b(today|tomorrow|yesterday|mon(?:day)?|tue(?:sday)?|wed(?:nesday)?|thu(?:rsday)?|fri(?:day)?|sat(?:urday)?|sun(?:day)?)\b`)

	if monthRegex.MatchString(text) {
		return true
	}
	if numericRegex.MatchString(text) {
		return true
	}
	if relativeRegex.MatchString(text) {
		return true
	}

	return false
}

// analyzeSentiment performs simple keyword-based sentiment analysis
func analyzeSentiment(text string) string {
	lowerText := strings.ToLower(text)

	// Pre-tokenization for efficiency
	words := strings.Fields(lowerText)
	wordMap := make(map[string]int)
	for _, w := range words {
		// Strip punctuation
		cleanWord := strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if cleanWord != "" {
			wordMap[cleanWord]++
		}
	}

	positiveWords := map[string]bool{
		"happy": true, "joy": true, "love": true, "excited": true, "great": true,
		"wonderful": true, "amazing": true, "fantastic": true, "excellent": true,
		"good": true, "better": true, "best": true, "grateful": true, "thankful": true,
		"blessed": true, "proud": true, "accomplished": true, "success": true, "win": true,
	}

	negativeWords := map[string]bool{
		"sad": true, "angry": true, "hate": true, "terrible": true, "awful": true,
		"bad": true, "worse": true, "worst": true, "depressed": true, "anxious": true,
		"worried": true, "stressed": true, "frustrated": true, "upset": true,
		"disappointed": true, "fail": true, "failure": true, "lost": true, "hurt": true, "pain": true,
	}

	positiveCount := 0
	negativeCount := 0

	// Single pass through dictionary
	for word, count := range wordMap {
		if positiveWords[word] {
			positiveCount += count
		}
		if negativeWords[word] {
			negativeCount += count
		}
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
	topics := []string{} // Initialize as empty slice, not nil
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
