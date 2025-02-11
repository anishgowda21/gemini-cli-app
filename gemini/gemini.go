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

type GeminiModel struct {
	DisplayName string
	ModelID     string
}

var supportedModels = []GeminiModel{
	{DisplayName: "Gemini Pro", ModelID: "models/gemini-pro"},
	{DisplayName: "Gemini 1.5 Pro", ModelID: "models/gemini-1.5-pro"},
	{DisplayName: "Gemini 2.0 Pro", ModelID: "models/gemini-2.0-pro-exp"},
	{DisplayName: "Gemini Pro Vision", ModelID: "models/gemini-pro-vision"},
	{DisplayName: "Gemini 1.5 Pro Latest", ModelID: "models/gemini-1.5-pro-latest"},
}

func initGeminiClient() (*genai.Client, context.Context, error) {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err)
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

func initializeChatHistory(chat *genai.ChatSession) {
	chat.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("You are a helpful Chatbot. Provide direct responses without any prefixes like 'REPLY:'. Keep responses short and precise."),
			},
			Role: roleUser,
		},
		{
			Parts: []genai.Part{
				genai.Text("Understood. I will provide direct responses without prefixes."),
			},
			Role: roleModel,
		},
	}
}

func streamOutput(text string) {
	for _, char := range text {
		fmt.Printf("%c", char)
		time.Sleep(30 * time.Millisecond)
	}
}

func processStream(iter *genai.GenerateContentResponseIterator, printOutput bool) (string, error) {
	var buffer strings.Builder

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
						response := string(text)
						if printOutput {
							streamOutput(response)
						}
						buffer.WriteString(response)
					}
				}
			}
		}
	}

	fullResponse := buffer.String()
	// Remove any REPLY: prefix that might appear
	fullResponse = strings.TrimPrefix(fullResponse, "REPLY:")
	fullResponse = strings.TrimSpace(fullResponse)

	fmt.Println()

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

	prompt := fmt.Sprintf(`You are a helpful chatbot. Respond to this message in two parts:

	1. A concise title (maximum 5 words) for this conversation
	2. Your response to the message

	Separate the two parts with "---". Do not add any labels or prefixes to either part.

	Message: %s`, userMessage)

	chat := model.StartChat()
	initializeChatHistory(chat)

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
		return "", "", fmt.Errorf("could not parse response format. Got: %s", fullResponse)
	}

	title = strings.TrimSpace(parts[0])
	reply = strings.TrimSpace(parts[1])

	// Remove any remaining prefixes
	title = strings.TrimPrefix(title, "TITLE:")
	title = strings.TrimPrefix(title, "Title:")
	reply = strings.TrimPrefix(reply, "REPLY:")
	reply = strings.TrimPrefix(reply, "Reply:")

	title = strings.TrimSpace(title)
	reply = strings.TrimSpace(reply)

	fmt.Print(agentPrefix)
	streamOutput(reply)
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

func ListModels() ([]GeminiModel, error) {
	return supportedModels, nil
}
