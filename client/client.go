package client

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func NewOpenAiClient() *openai.Client {
	client := openai.NewClient(
		option.WithBaseURL("http://172.26.208.1:1234/v1"),
	)
	return &client
}
