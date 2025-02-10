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
		fmt.Println("3) Update existing conversation (Add message)")
		fmt.Println("4) List available models") // New option
		fmt.Println("5) Exit")
		choice := readInput("Enter choice: ")

		switch choice {
		case "1":
			listConversations()
		case "2":
			startNewConversation()
		case "3":
			updateConversation()
		case "4":
			models, err := gemini.ListModels()
			if err != nil {
				fmt.Println("Error listing models:", err)
			} else {
				fmt.Println("\nAvailable Models:")
				for _, model := range models {
					fmt.Println(model)
				}
			}
		case "5":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}
