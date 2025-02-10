package gemini

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gemini-cli-app/database"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	roleUser    = "user"
	roleModel   = "model"
	agentPrefix = "Agent: "
)

func initGeminiClient() (*genai.Client, context.Context, error) {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err) // Continue even on error.
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	return client, ctx, nil
}

// configureModel sets up common configuration for the model
func configureModel(model *genai.GenerativeModel) {
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
}

// initializeChatHistory sets up the initial chat history
func initializeChatHistory(chat *genai.ChatSession) {
	chat.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("You are a helpful Chatbot, that helps users by answring their questions. The responses should be short, and precise."),
			},
			Role: roleUser,
		},
		{
			Parts: []genai.Part{
				genai.Text("Understood. I will do my best to be helpful!"),
			},
			Role: roleModel,
		},
	}
}

// processStream handles streaming responses from the model
func processStream(iter *genai.GenerateContentResponseIterator, printOutput bool) (string, error) {
	fullResponse := ""
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
						if printOutput {
							fmt.Print(string(text))
						}
						fullResponse += string(text)
					}
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fullResponse, nil
}

func StartConversation(modelName, userMessage string) (title, reply string, err error) {
	client, ctx, err := initGeminiClient()
	if err != nil {
		return "", "", err
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	configureModel(model)

	prompt := fmt.Sprintf(`You are a helpful chatbot. I will provide an initial message. Respond with two parts, separated by "---":

	1.  TITLE: A concise title (maximum 5 words) for this conversation.
	2.  REPLY: A response to my initial message.

	Initial message: %s`, userMessage)

	chat := model.StartChat()
	initializeChatHistory(chat)

	// Use SendMessage instead of SendMessageStream for single response
	resp, err := chat.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return "", "", fmt.Errorf("error sending message: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", "", fmt.Errorf("no response candidates received")
	}

	var fullResponse string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			fullResponse += string(text)
		}
	}

	parts := strings.SplitN(fullResponse, "---", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("could not parse title and reply.  Expected format 'TITLE---REPLY', got: %s", fullResponse)
	}

	title = strings.TrimSpace(parts[0])
	reply = strings.TrimSpace(parts[1])

	// Print reply with streaming effect
	fmt.Print(agentPrefix)
	for _, r := range reply {
		fmt.Printf("%c", r)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println()

	return title, reply, nil
}

func GenerateReply(modelName string, messages []database.Message) (string, error) {
	client, ctx, err := initGeminiClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	model := client.GenerativeModel(modelName)
	configureModel(model)

	chat := model.StartChat()
	initializeChatHistory(chat)

	// Add message history
	for _, msg := range messages[:len(messages)-1] {
		role := roleUser
		if msg.Role == "assistant" {
			role = roleModel
		}
		chat.History = append(chat.History, &genai.Content{
			Parts: []genai.Part{genai.Text(msg.RawContent)},
			Role:  role,
		})
	}

	if len(messages) == 0 {
		return "", fmt.Errorf("no messages in conversation")
	}

	fmt.Print(agentPrefix)
	iter := chat.SendMessageStream(ctx, genai.Text(messages[len(messages)-1].RawContent))
	return processStream(iter, true)
}

func ListModels() ([]string, error) {
	client, ctx, err := initGeminiClient()
	if err != nil {
		return nil, err
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
		modelNames = append(modelNames, model.Name)
	}

	return modelNames, nil
}
