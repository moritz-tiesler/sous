package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ollama/ollama/api"
)

func main() {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	agent := NewAgent(client, getUserMessage)
	err = agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
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

func (a *Agent) Run(ctx context.Context) error {
	conversation := []api.Message{}
	fmt.Println("Chat with Devstral (use 'ctrl-c' to quit)")

	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			break
		}

		userMessage := api.Message{
			Role:    "user",
			Content: userInput,
		}
		conversation = append(conversation, userMessage)

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}
		conversation = append(conversation, message)

		// for _, content := range message.Content {
		// 	switch content.(type) {
		// 	case "text":
		// 		fmt.Printf("\u001b[93mClaude\u001b[0m: %s\n", content.Text)
		// 	}
		// }

		fmt.Printf("\u001b[93mDevstral\u001b[0m: %s\n", message.Content)
	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []api.Message) (api.Message, error) {
	req := &api.ChatRequest{
		// Model:  "gemma2",
		Model:    "devstral:24b-small-2505-q8_0",
		Messages: conversation,
		Stream:   new(bool),
	}
	var message api.Message
	respFunc := func(cr api.ChatResponse) error {
		message = cr.Message
		return nil
	}
	err := a.client.Chat(ctx, req, respFunc)
	return message, err
}
