package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/agentbill/agentbill-go"
)

func main() {
	// Initialize AgentBill with your API key
	client := agentbill.Init(agentbill.Config{
		APIKey:     getEnv("AGENTBILL_API_KEY", "your-api-key"),
		BaseURL:    getEnv("AGENTBILL_BASE_URL", ""),
		CustomerID: "customer-123",
		Debug:      true,
	})

	// Wrap your OpenAI client
	openai := client.WrapOpenAI()

	// Use OpenAI normally - tracking is automatic!
	ctx := context.Background()
	response, err := openai.ChatCompletion(ctx, "gpt-4o-mini", []map[string]string{
		{"role": "system", "content": "You are a helpful assistant."},
		{"role": "user", "content": "What is the capital of France?"},
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(response)

	// All usage (tokens, cost, latency) is automatically tracked to your AgentBill dashboard
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
