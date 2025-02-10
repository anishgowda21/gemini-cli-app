package main

import (
	"fmt"
	"gemini-cli-app/gemini"
)

func StartNewConversation() {
	availableModels, err := gemini.ListModels()

	if err != nil {
		fmt.Println("Error listing models:", err)
		return
	}

	fmt.Println("\nAvailable Models:")

	for i, modelName := range availableModels {
		fmt.Printf("%d) %s\n", i+1, modelName)
	}

	modelChoice := -1
}
