package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-redis/redis/v8"
	"github.com/google/generative-ai-go/genai"
	"github.com/mindgarden/server/internal/observability"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

type Service struct {
	geminiClient *genai.Client
	groqClient   *openai.Client
	redisClient  *redis.Client
	embedModel   string
	embedDims    int
}

func NewService() *Service {
	ctx := context.Background()

	// Gemini Setup
	geminiKey := os.Getenv("GEMINI_API_KEY")
	gClient, err := genai.NewClient(ctx, option.WithAPIKey(geminiKey))
	if err != nil {
		fmt.Printf("Error creating Gemini client: %v\n", err)
	}

	// Groq Setup
	groqKey := os.Getenv("GROQ_API_KEY")
	groqConfig := openai.DefaultConfig(groqKey)
	groqConfig.BaseURL = "https://api.groq.com/openai/v1"
	oClient := openai.NewClientWithConfig(groqConfig)

	// Redis Setup
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	// Check if URL starts with redis://, if so parse it, else treat as addr
	var rOptions *redis.Options
	if len(redisURL) > 8 && redisURL[:8] == "redis://" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			fmt.Printf("Error parsing REDIS_URL: %v\n", err)
			rOptions = &redis.Options{Addr: "localhost:6379"}
		} else {
			rOptions = opt
		}
	} else {
		rOptions = &redis.Options{Addr: redisURL}
	}

	rClient := redis.NewClient(rOptions)

	// Test Redis connection
	if err := rClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("Warning: Failed to connect to Redis: %v. Caching disabled.\n", err)
		// We keep rClient but maybe it will fail on calls.
		// Or we can set it to nil and check before use.
		// For now, let's keep it, maybe it comes up later.
	} else {
		fmt.Println("Redis connection established")
	}

	embedDims := 768
	if raw := strings.TrimSpace(os.Getenv("EMBEDDING_DIMENSIONS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			embedDims = parsed
		}
	}

	return &Service{
		geminiClient: gClient,
		groqClient:   oClient,
		redisClient:  rClient,
		embedModel:   os.Getenv("GEMINI_EMBEDDING_MODEL"),
		embedDims:    embedDims,
	}
}

func buildEmbeddingModelCandidates(configured string) []string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		configured = "gemini-embedding-001"
	}

	candidates := []string{configured}

	if strings.HasPrefix(configured, "models/") {
		trimmed := strings.TrimPrefix(configured, "models/")
		if trimmed != "" {
			candidates = append(candidates, trimmed)
		}
	} else {
		candidates = append(candidates, "models/"+configured)
	}

	candidates = append(candidates, "gemini-embedding-001", "models/gemini-embedding-001")

	uniq := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		uniq = append(uniq, candidate)
	}
	return uniq
}

func normalizeEmbeddingDimensions(values []float32, target int) []float32 {
	if target <= 0 {
		return values
	}
	if len(values) == target {
		return values
	}
	if len(values) > target {
		return values[:target]
	}

	out := make([]float32, target)
	copy(out, values)
	return out
}

func (s *Service) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Check Cache
	if s.redisClient != nil {
		hash := sha256.Sum256([]byte(text))
		hashStr := hex.EncodeToString(hash[:])
		cacheKey := fmt.Sprintf("emb:%d:%s", s.embedDims, hashStr)

		val, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var embedding []float32
			if err := json.Unmarshal([]byte(val), &embedding); err == nil {
				return normalizeEmbeddingDimensions(embedding, s.embedDims), nil
			}
		} else if err != redis.Nil {
			// Log error but continue
			fmt.Printf("Redis error: %v\n", err)
		}
	}

	if s.geminiClient == nil {
		return nil, errors.New("gemini client not initialized")
	}

	modelCandidates := buildEmbeddingModelCandidates(s.embedModel)
	var lastErr error
	var values []float32

	for _, modelName := range modelCandidates {
		em := s.geminiClient.EmbeddingModel(modelName)
		res, err := em.EmbedContent(ctx, genai.Text(text))
		if err != nil {
			lastErr = err
			continue
		}
		if res == nil || res.Embedding == nil || len(res.Embedding.Values) == 0 {
			lastErr = errors.New("no embedding returned")
			continue
		}
		values = res.Embedding.Values
		break
	}

	if len(values) == 0 {
		if lastErr == nil {
			lastErr = errors.New("all embedding attempts failed")
		}
		return nil, fmt.Errorf(
			"gemini embedding error (tried models: %s): %w",
			strings.Join(modelCandidates, ", "),
			lastErr,
		)
	}

	observability.TrackEmbeddingCost(utf8.RuneCountInString(text))

	// Cache Result
	values = normalizeEmbeddingDimensions(values, s.embedDims)

	if s.redisClient != nil {
		hash := sha256.Sum256([]byte(text))
		hashStr := hex.EncodeToString(hash[:])
		cacheKey := fmt.Sprintf("emb:%d:%s", s.embedDims, hashStr)

		data, err := json.Marshal(values)
		if err == nil {
			s.redisClient.Set(ctx, cacheKey, data, 24*time.Hour)
		}
	}

	return values, nil
}

func (s *Service) GenerateAnswer(ctx context.Context, prompt string) (string, error) {
	if s.groqClient == nil {
		return "", errors.New("groq client not initialized")
	}

	// PROMPT ENGINEERING:
	// Groq LLaMA-3 models work best with explicit system prompts.
	resp, err := s.groqClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: "llama-3.1-8b-instant",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful assistant for Mind Garden. Answer the user's question based strictly on the context provided.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: 0.1, // Grounded
	})

	if err != nil {
		return "", fmt.Errorf("groq generation error: %w", err)
	}

	observability.TrackLLMCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	return resp.Choices[0].Message.Content, nil
}

// MetadataAnalysis holds the results of chunk metadata analysis
type MetadataAnalysis struct {
	Sentiment string
	Topics    []string
}

// AnalyzeChunkMetadata uses Gemini to analyze sentiment and extract topics from text
func (s *Service) AnalyzeChunkMetadata(ctx context.Context, text string) (*MetadataAnalysis, error) {
	if s.geminiClient == nil {
		return nil, errors.New("gemini client not initialized")
	}

	// Use Gemini Flash Latest (confirmed available model)
	model := s.geminiClient.GenerativeModel("gemini-flash-latest")

	// Configure for structured output
	model.SetTemperature(0.1) // Low temperature for consistent results

	prompt := fmt.Sprintf(`Analyze the following journal entry chunk and provide:
1. Sentiment (one word: positive, negative, or neutral)
2. Topics (up to 5 key topics/themes as single words or short phrases)

Text: %s

Respond in this exact format:
Sentiment: <sentiment>
Topics: <topic1>, <topic2>, <topic3>

If there are no clear topics, respond with "Topics: none"`, text)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini analysis error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("no response from gemini")
	}

	// Parse the response
	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return parseMetadataResponse(responseText)
}

// parseMetadataResponse parses the Gemini response into structured metadata
func parseMetadataResponse(response string) (*MetadataAnalysis, error) {
	result := &MetadataAnalysis{
		Sentiment: "neutral",
		Topics:    []string{},
	}

	// Simple parsing - look for "Sentiment:" and "Topics:"
	sentimentPrefix := "Sentiment:"
	topicsPrefix := "Topics:"

	// Find sentiment
	if idx := findSubstring(response, sentimentPrefix); idx != -1 {
		// Extract sentiment value
		start := idx + len(sentimentPrefix)
		end := findNextNewline(response, start)
		if end == -1 {
			end = len(response)
		}
		sentiment := trimSpace(response[start:end])
		sentiment = toLowerCase(sentiment)

		// Validate sentiment
		if sentiment == "positive" || sentiment == "negative" || sentiment == "neutral" {
			result.Sentiment = sentiment
		}
	}

	// Find topics
	if idx := findSubstring(response, topicsPrefix); idx != -1 {
		start := idx + len(topicsPrefix)
		end := findNextNewline(response, start)
		if end == -1 {
			end = len(response)
		}
		topicsStr := trimSpace(response[start:end])

		if topicsStr != "" && topicsStr != "none" {
			// Split by comma
			topics := splitByComma(topicsStr)
			for _, topic := range topics {
				topic = trimSpace(topic)
				if topic != "" && topic != "none" {
					result.Topics = append(result.Topics, topic)
				}
			}
		}
	}

	return result, nil
}

// Helper functions for string parsing
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func findNextNewline(s string, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			result[i] = s[i] + 32
		} else {
			result[i] = s[i]
		}
	}
	return string(result)
}

func splitByComma(s string) []string {
	var result []string
	current := ""

	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
