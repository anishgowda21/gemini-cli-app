package gemini

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gemini-cli-app/database"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func generateReply(modelName string, messages []database.Message) (string, error) {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")

	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))

	if err != nil {
		return "", fmt.Errorf("failed to create genai client: %w", err)
	}

	defer client.Close()

	model := client.GenerativeModel(modelName)

	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockLowAndAbove,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockLowAndAbove,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockLowAndAbove,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockLowAndAbove,
		},
	}

	config := &genai.GenerationConfig{}
	config.SetMaxOutputTokens(2048)
	config.SetTemperature(0.7)
	config.SetTopP(0.9)

	model.GenerationConfig = *config

	chat := model.StartChat()

	chat.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("You are a helpful Chatbot, that helps users by answring their questions.  The responses should be short, and precise."),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Understood. I will do my best to be helpful!"),
			},
			Role: "model",
		},
	}

	for _, msg := range messages[:len(messages)-1] {
		role := "user"

		if msg.Role == "assistant" {
			role = "model" // Gemini uses "model" for the assistant.
		}
		chat.History = append(chat.History, &genai.Content{
			Parts: []genai.Part{
				genai.Text(msg.RawContent),
			},
			Role: role,
		})
	}

	var iter *genai.GenerateContentResponseIterator
	if len(messages) > 0 {
		iter = chat.SendMessageStream(ctx, genai.Text(messages[len(messages)-1].RawContent))
	} else {
		return "", fmt.Errorf("no messages in conversation")
	}

	fullResponse := ""

	fmt.Print("Agent: ")
	for {
		resp, err := iter.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			if err.Error() == "context canceled" {
				return "", fmt.Errorf("Stream interrupted: %v", err)
			}
			return "", fmt.Errorf("error during streaming: %w", err)
		}

		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					if text, ok := part.(genai.Text); ok {
						fmt.Print(string(text))

						fullResponse += string(text)
					}
				}
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println()
	return fullResponse, nil
}

func ListModels() ([]string, error) {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err) // Continue even on error.
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}
	defer client.Close()

	var modelNames []string

	iter := client.ListModels(ctx)

	for {
		model, err := iter.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error listing models: %w", err)
		}

		modelNames = append(modelNames, model.DisplayName)
	}

	return modelNames, nil
}
