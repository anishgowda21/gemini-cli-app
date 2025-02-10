package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func readInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Error reading input:", err)
		return ""
	}
	return strings.TrimSpace(input)
}
