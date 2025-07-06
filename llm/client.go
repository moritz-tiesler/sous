package llm

import (
	"context"

	"github.com/ollama/ollama/api"
)

type ChatClient interface {
	Chat(ctx context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error
}

func NewOllamaClient() (ChatClient, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}
	return client, nil
}
