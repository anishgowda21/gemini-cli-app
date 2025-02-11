package main

import (
	"fmt"
	"gemini-cli-app/database"
	"gemini-cli-app/gemini"
	"log"
)

func main() {
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	for {
		fmt.Println("\n=== GEMINI CLI Chat App ===")
		fmt.Println("1) List conversations")
		fmt.Println("2) Start new conversation")
		fmt.Println("3) Continue a conversation")
		fmt.Println("4) List available models")
		fmt.Println("5) Delete conversation")
		fmt.Println("6) Exit")
		choice := readInput("Enter choice: ")

		switch choice {
		case "1":
			listConversations()
		case "2":
			startNewConversation()
		case "3":
			continueConversation()
		case "4":
			models, err := gemini.ListModels()
			if err != nil {
				fmt.Println("Error listing models:", err)
			} else {
				fmt.Println("\nAvailable Models:")
				for _, model := range models {
					fmt.Println(model.DisplayName)
				}
			}
		case "5":
			deleteConversation()
		case "6":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}
