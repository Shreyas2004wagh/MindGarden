package llm

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

type Service struct {
	geminiClient *genai.Client
	groqClient   *openai.Client
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

	return &Service{
		geminiClient: gClient,
		groqClient:   oClient,
	}
}

func (s *Service) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if s.geminiClient == nil {
		return nil, errors.New("gemini client not initialized")
	}
	// Use text-embedding-004 for better performance/quality
	em := s.geminiClient.EmbeddingModel("text-embedding-004")
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("gemini embedding error: %w", err)
	}
	if res.Embedding == nil {
		return nil, errors.New("no embedding returned")
	}
	return res.Embedding.Values, nil
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
	if len(resp.Choices) == 0 {
		return "", errors.New("groq generation returned no choices")
	}

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
