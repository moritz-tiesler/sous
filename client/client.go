package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type ChatContext struct {
	ctx    context.Context
	Cancel context.CancelFunc
}

type Client struct {
	c           *openai.Client
	modelName   string
	mu          sync.Mutex
	ChatContext *ChatContext
}

func (c *Client) SetActiveChatContext(ctx context.Context, cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ChatContext = &ChatContext{ctx: ctx, Cancel: cancel}
}

func (c *Client) ClearChatContext() {
	c.SetActiveChatContext(context.TODO(), nil)
}

// todo inherit context, ie pass the context to New
func New(modelName string) *Client {
	c := openai.NewClient(
		option.WithBaseURL("http://172.26.208.1:1234/v1/"),
	)
	return &Client{
		c:           &c,
		modelName:   modelName,
		ChatContext: &ChatContext{},
	}
}

func (c *Client) RunInference(
	ctx context.Context,
	conversation []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolParam,
) (openai.ChatCompletionMessage, error) {
	reqCtx, reqCancel := context.WithCancel(ctx)
	c.SetActiveChatContext(reqCtx, reqCancel)
	chatCompletion, err := c.c.Chat.Completions.New(reqCtx, openai.ChatCompletionNewParams{
		Messages: conversation,
		Model:    c.modelName,
		Tools:    tools,
	})

	var message openai.ChatCompletionMessage
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return message, fmt.Errorf("inference cancelled: %v", err)
		}
		return message, err
	}
	c.ClearChatContext()
	return chatCompletion.Choices[0].Message, nil
}

func (c *Client) RunInferenceSingle(
	ctx context.Context,
	prompt string,
) (string, error) {
	reqCtx, reqCancel := context.WithCancel(ctx)
	c.SetActiveChatContext(reqCtx, reqCancel)
	completion, err := c.c.Completions.New(reqCtx, openai.CompletionNewParams{
		Prompt: openai.CompletionNewParamsPromptUnion{
			OfString: openai.String(prompt),
		},

		Model: openai.CompletionNewParamsModel(c.modelName),
	})

	var message string
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return message, fmt.Errorf("inference cancelled: %v", err)
		}
		return message, err
	}
	c.ClearChatContext()
	return completion.Choices[0].Text, nil
}
