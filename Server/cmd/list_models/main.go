package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("📋 Available Gemini Models:")
	fmt.Println("=" + "===========================================")

	iter := client.ListModels(ctx)
	for {
		model, err := iter.Next()
		if err != nil {
			break
		}

		fmt.Printf("\n🔹 Model: %s\n", model.Name)
		fmt.Printf("   Display Name: %s\n", model.DisplayName)
		fmt.Printf("   Description: %s\n", model.Description)

		// Check supported methods
		fmt.Printf("   Supported Methods: ")
		for i, method := range model.SupportedGenerationMethods {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(method)
		}
		fmt.Println()
	}

	fmt.Println("\n" + "=" + "===========================================")
	fmt.Println("✅ Use one of the model names above in your code")
}
