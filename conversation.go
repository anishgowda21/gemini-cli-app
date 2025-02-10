package main

import (
	"fmt"
	"gemini-cli-app/database"
	"gemini-cli-app/gemini"
	"strconv"
	"strings"
)

// Helper function for getting valid numeric input
func getValidChoice(prompt string, maxChoice int) (int, error) {
    for {
        choiceStr := readInput(prompt)
        choice, err := strconv.Atoi(choiceStr)
        if err != nil || choice < 1 || choice > maxChoice {
            fmt.Printf("Invalid choice. Please enter a number between 1 and %d.\n", maxChoice)
            continue
        }
        return choice, nil
    }
}

func conversationLoop(convoID, modelName string) {
    for {
        userMsg := readInput("Enter your message (type /bye to exit): ")
        if userMsg == "/bye" {
            break
        }

        // Group database operations
        if err := handleMessageExchange(convoID, modelName, userMsg); err != nil {
            fmt.Println("Error in conversation:", err)
            continue
        }
    }
}

// handleMessageExchange handles the full cycle of message exchange
func handleMessageExchange(convoID, modelName, userMsg string) error {
    if err := database.AddMessage(convoID, "user", userMsg); err != nil {
        return fmt.Errorf("adding user message: %w", err)
    }

    messages, err := database.GetMessagesByConversationID(convoID)
    if err != nil {
        return fmt.Errorf("retrieving messages: %w", err)
    }

    reply, err := gemini.GenerateReply(modelName, messages)
    if err != nil {
        return fmt.Errorf("generating reply: %w", err)
    }

    if err := database.AddMessage(convoID, "assistant", reply); err != nil {
        return fmt.Errorf("adding assistant message: %w", err)
    }

    if err := database.UpdateConversation(convoID); err != nil {
        return fmt.Errorf("updating conversation: %w", err)
    }

    return nil
}

func startNewConversation() {
    availableModels, err := gemini.ListModels()
    if err != nil {
        fmt.Println("Error listing models:", err)
        return
    }

    fmt.Println("\nAvailable Models:")
    for i, modelName := range availableModels {
        fmt.Printf("%d) %s\n", i+1, modelName)
    }

    modelChoice, err := getValidChoice("Enter model choice number: ", len(availableModels))
    if err != nil {
        fmt.Println("Error selecting model:", err)
        return
    }

    selectedModel := availableModels[modelChoice-1]
    initialMessage := readInput("Start Your Conversation: ")

    title, reply, err := gemini.StartConversation(selectedModel, initialMessage)
    if err != nil {
        fmt.Println("Error starting conversation:", err)
        return
    }

    convoID, err := database.CreateConversation(title, selectedModel)
    if err != nil {
        fmt.Println("Error creating conversation:", err)
        return
    }

    // Initialize conversation with first messages
    if err := initializeConversation(convoID, initialMessage, reply); err != nil {
        fmt.Println("Error initializing conversation:", err)
        return
    }

    fmt.Printf("New conversation created with ID: %s and title: %s\n", convoID, title)
    conversationLoop(convoID, selectedModel)
}

func initializeConversation(convoID, userMsg, reply string) error {
    if err := database.AddMessage(convoID, "user", userMsg); err != nil {
        return fmt.Errorf("adding initial user message: %w", err)
    }

    if err := database.AddMessage(convoID, "assistant", reply); err != nil {
        return fmt.Errorf("adding initial assistant message: %w", err)
    }

    return nil
}

func updateConversation() {
    convos, err := database.ListConversations()
    if err != nil {
        fmt.Println("Error retrieving conversations:", err)
        return
    }

    if len(convos) == 0 {
        fmt.Println("No conversations found.")
        return
    }

    fmt.Println("\nSelect a conversation to update:")
    for i, convo := range convos {
        fmt.Printf("%d) %s (Model: %s, ID: %s)\n", i+1, convo.Title, convo.Model, convo.ID)
    }

    choice, err := getValidChoice("Enter choice number: ", len(convos))
    if err != nil {
        fmt.Println("Error selecting conversation:", err)
        return
    }

    selectedConvo := convos[choice-1]
    conversationLoop(selectedConvo.ID, selectedConvo.Model)
}

func listConversations() {
    convos, err := database.ListConversations()
    if err != nil {
        fmt.Println("Error retrieving conversations:", err)
        return
    }

    if len(convos) == 0 {
        fmt.Println("No conversations found.")
        return
    }

    fmt.Println("\nConversations:")
    for i, convo := range convos {
        fmt.Printf("%d) %s (ID: %s)\n", i+1, convo.Title, convo.ID)
    }

    for {
        viewChoice := readInput("Do you want to view messages of a conversation? (y/n): ")
        viewChoice = strings.ToLower(viewChoice)

        switch viewChoice {
        case "y":
            choice, err := getValidChoice("Enter choice number: ", len(convos))
            if err != nil {
                fmt.Println("Error selecting conversation:", err)
                return
            }
            showConversationMessages(convos[choice-1].ID)
            return
        case "n":
            return
        default:
            fmt.Println("Invalid choice. Please enter 'y' or 'n'.")
        }
    }
}

func showConversationMessages(convoID string) {
    messages, err := database.GetMessagesByConversationID(convoID)
    if err != nil {
        fmt.Println("Error retrieving messages:", err)
        return
    }

    if len(messages) == 0 {
        fmt.Println("No messages found for this conversation.")
        return
    }

    fmt.Printf("\nMessages for conversation %s:\n", convoID)
    for _, msg := range messages {
        fmt.Printf("[%s] %s: %s\n", msg.CreatedAt.Format("15:04:05"), msg.Role, msg.Content)
    }
}
