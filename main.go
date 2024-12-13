package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func logWithTimestamp(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	log.Printf("[%s] %s", timestamp, fmt.Sprintf(format, args...))
}

func main() {
	apiKey := flag.String("api-key", "", "API key for OpenAI")
	systemPromptFile := flag.String("system-prompt", "", "File path for the system prompt")
	userPromptFile := flag.String("user-prompt", "", "File path for the user prompt (optional)")
	aiModel := flag.String("ai-model", "gpt-4o-mini", "AI model to use (default: gpt-4o-mini)")
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("API key is missing")
	}

	if *systemPromptFile == "" {
		log.Fatal("System prompt file is missing")
	}

	systemPromptContent, err := ioutil.ReadFile(*systemPromptFile)
	if err != nil {
		log.Fatalf("Error reading system prompt file: %v", err)
	}

	var userPromptContent []byte
	if *userPromptFile != "" {
		userPromptContent, err = ioutil.ReadFile(*userPromptFile)
		if err != nil {
			log.Fatalf("Error reading user prompt file: %v", err)
		}
	}

	cmd := exec.Command("xclip", "-o")
	inputTextBytes, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	inputText := string(inputTextBytes)
	logWithTimestamp("Detected clipboard text: %s", inputText)

	messages := []map[string]interface{}{
		{
			"role":    "system",
			"content": string(systemPromptContent),
		},
	}

	if len(userPromptContent) > 0 {
		messages = append([]map[string]interface{}{
			{
				"role":    "user",
				"content": string(userPromptContent),
			},
		}, messages...)
	}

	messages = append(messages, map[string]interface{}{
		"role":    "user",
		"content": fmt.Sprintf("Process the following text: %s", inputText),
	})

	requestBody := map[string]interface{}{
		"model":    *aiModel,
		"messages": messages,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+*apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		log.Fatal(err)
	}

	choices := result["choices"].([]interface{})
	responseText := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	logWithTimestamp("AI response text: %s", responseText)

	cmd = exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(responseText)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command("xdotool", "key", "ctrl+v")
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
