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
	
	return resp.Choices[0].Message.Content, nil
}
