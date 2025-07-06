package llm

import (
	"context"
	"io"

	"github.com/moritz-tiesler/sous/tools"
	"github.com/ollama/ollama/api"
)

type Message struct {
	Role      string
	Content   string
	ToolCalls []ToolCall
}

type ToolCall struct {
	ID       string
	Type     string
	Function Function
}

type Function struct {
	Name      string
	Arguments string
}

type ChatClient interface {
	Chat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error)
}

type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []tools.Tool
}

func NewOllamaClient() (ChatClient, error) {
	return &ollamaClient{}, nil
}

type ollamaClient struct{}

func (c *ollamaClient) Chat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	ollamaReq := &api.ChatRequest{
		Model:    req.Model,
		Messages: toOllamaMessages(req.Messages),
		Tools:    toOllamaTools(req.Tools),
	}

	// This is a simplified example. In a real implementation, you would
	// handle the streaming response and adapt it to the io.ReadCloser interface.
	// For now, we'll just pass the request to the client.
	// The actual response handling will be implemented in a later step.
	_, err = client.Chat(ctx, ollamaReq, func(api.ChatResponse) error {
		return nil
	})
	if err != nil {
		return nil, err
	}

	return io.NopCloser(strings.NewReader("")), nil
}

func toOllamaMessages(messages []Message) []api.Message {
	var ollamaMessages []api.Message
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return ollamaMessages
}

func toOllamaTools(tools []tools.Tool) api.Tools {
	var ollamaTools api.Tools
	for _, tool := range tools {
		ollamaTools = append(ollamaTools, api.Tool{
			Type:     tool.Type,
			Function: toOllamaFunction(tool.Function),
		})
	}
	return ollamaTools
}

func toOllamaFunction(function tools.Function) api.ToolFunction {
	return api.ToolFunction{
		Name:        function.Name,
		Description: function.Description,
		Parameters:  toOllamaParameters(function.Parameters),
	}
}

func toOllamaParameters(params tools.ToolFunctionParameters) map[string]interface{} {
	// This is a simplified conversion. A real implementation would need to
	// handle the full complexity of the parameters.
	return map[string]interface{}{
		"type":       params.Type,
		"properties": params.Properties,
		"required":   params.Required,
	}
}