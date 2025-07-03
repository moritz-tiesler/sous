package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

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

const PREFIX = "\u001b[93mDevstral\u001b[0m: %s"

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

		stream := true
		message, err := a.runInference(ctx, conversation, stream)
		if err != nil {
			return err
		}
		conversation = append(conversation, message)

	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []api.Message, stream bool) (api.Message, error) {
	req := &api.ChatRequest{
		// Model:  "gemma2",
		Model:    "devstral:24b-small-2505-q8_0",
		Messages: conversation,
		Stream:   &stream,
	}
	content := strings.Builder{}
	message := api.Message{}
	respFunc := func(cr api.ChatResponse) error {
		if stream {
			if !cr.Done {
				fmt.Printf("%s", cr.Message.Content)
			} else {
				fmt.Println()
			}
		}
		content.WriteString(cr.Message.Content)
		return nil
	}

	fmt.Printf(PREFIX, "")
	err := a.client.Chat(ctx, req, respFunc)
	message.Content = content.String()
	return message, err
}
