package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ollama/ollama/api"
)

func main() {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	req := &api.ChatRequest{
		// Model:  "gemma2",
		Model: "devstral:24b-small-2505-q8_0",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "how many planets are there?",
			},
		},

		// set streaming to false
		Stream: new(bool),
	}

	ctx := context.Background()
	respFunc := func(resp api.ChatResponse) error {
		// Only print the response here; GenerateResponse has a number of other
		// interesting fields you want to examine.
		fmt.Println(resp.Message.Content)
		return nil
	}

	// scanner := bufio.NewScanner(os.Stdin)
	// getUserMessage := func() (string, bool) {
	// 	if !scanner.Scan() {
	// 		return "", false
	// 	}
	// 	return scanner.Text(), true
	// }

	err = client.Chat(ctx, req, respFunc)
	if err != nil {
		log.Fatal(err)
	}

	// agent := NewAgent(client, getUserMessage)
}

func NewAgent(client *api.Client, getUserMessage func() (string, bool)) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
	}
}

type Agent struct {
	client         *api.Client
	getUserMessage func() (string, bool)
}

// func (a *Agent) Run(ctx context.Context) error {
// 	conversation := []
// }
