package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/moritz-tiesler/sous/llm"
	"github.com/moritz-tiesler/sous/output"
	"github.com/moritz-tiesler/sous/tools"
)

type ChatContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Agent struct {
	client         llm.ChatClient
	getUserMessage func() (string, bool)
	toolDefs       []tools.Tool
	toolMap        map[string]func(map[string]interface{}) (string, error)

	mu          sync.Mutex
	chatContext *ChatContext
}

func NewAgent(
	client llm.ChatClient,
	getUserMessage func() (string, bool),
) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		toolDefs:       tools.Tools(),
		toolMap:        tools.ToolMap(),
		chatContext:    &ChatContext{context.Background(), nil},
	}
}

func (a *Agent) SetActiveChatContext(ctx context.Context, cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chatContext = &ChatContext{ctx: ctx, cancel: cancel}
}

func (a *Agent) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.chatContext.cancel != nil {
		a.chatContext.cancel()
		fmt.Println()
		a.chatContext = &ChatContext{context.Background(), nil}
	}
}

func (a *Agent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chatContext.cancel != nil
}

const PREFIX = "\u001b[93mSous\u001b[0m: %s"

func (a *Agent) Run(ctx context.Context) error {
	conversation := []llm.Message{}
	fmt.Println("Chat with Sous")

	readUserInput := true
	for {
		if len(conversation) > 10 {
			output.PrintAction("%s...\n", "SUMMARIZING")
			summary, err := a.summarizeConvo(ctx, conversation)
			if err != nil {
				return err
			}

			conversation = append([]llm.Message{}, summary)
			output.PrintAction("NEW CONVO LEN=%d...\n", len(conversation))
			output.PrintAction("NEW CONVO STarts with=%s...\n", summary.Content)
		}
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMessage := llm.Message{
				Role:    "user",
				Content: userInput,
			}
			conversation = append(conversation, userMessage)
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			fmt.Println(dumpConvo(conversation))
			return err
		}
		conversation = append(conversation, message)

		toolResults := map[string]string{}
		for _, toolCall := range message.ToolCalls {
			f := toolCall.Function
			result, err := a.executeTool(f.Name, f.Arguments)
			var res string
			res += result
			if err != nil {
				res += fmt.Sprintf("error: %s\n", err.Error())
			}
			toolResults[f.Name] = res
		}
		if len(toolResults) == 0 {
			readUserInput = true
			go func() {
				if err = ping(); err != nil {
					panic(err.Error())
				}
			}()
			continue
		}

		readUserInput = false
		toolResMessage := fmt.Sprintf("%v", toolResults)
		conversation = append(conversation, llm.Message{Role: "tool", Content: toolResMessage})

	}
	return nil
}

func (a *Agent) summarizeConvo(ctx context.Context, conversation []llm.Message) (llm.Message, error) {
	reqCtx, reqCancel := context.WithCancel(ctx)
	a.SetActiveChatContext(reqCtx, reqCancel)
	defer a.SetActiveChatContext(context.TODO(), nil)

	conversation = append(conversation, llm.Message{
		Role:    "user",
		Content: "please summarize the active conversation, so that you can pick up your work form here. include the original user instructions so that you do not loose the context of the task at hand. include previous tool calls in this summary.",
	})

	req := &llm.ChatRequest{
		Model:    "qwen3:14b_devstral",
		Messages: conversation,
		Tools:    a.toolDefs,
	}

	resp, err := a.client.Chat(reqCtx, req)
	if err != nil {
		return llm.Message{}, err
	}
	defer resp.Close()

	content, err := io.ReadAll(resp)
	if err != nil {
		return llm.Message{}, err
	}

	return llm.Message{
		Role:    "user",
		Content: string(content),
	}, nil
}

func (a *Agent) executeTool(name string, args string) (string, error) {
	var toolFunc func(map[string]interface{}) (string, error)
	var found bool
	for n, f := range a.toolMap {
		if n == name {
			toolFunc = f
			found = true
			break
		}
	}
	if !found {
		return fmt.Sprintf("tool '%s' not found", name), nil
	}

	var arguments map[string]interface{}
	err := json.Unmarshal([]byte(args), &arguments)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	response, err := toolFunc(arguments)
	output.PrintAction("tool: %s, %v\n%v\n", name, args, response)
	if err != nil {
		output.PrintAction("errors %s %v\n", response, err.Error())
		return response, err
	}
	return response, nil
}

func (a *Agent) runInference(
	ctx context.Context,
	conversation []llm.Message,
) (llm.Message, error) {
	reqCtx, reqCancel := context.WithCancel(ctx)
	a.SetActiveChatContext(reqCtx, reqCancel)
	defer a.SetActiveChatContext(context.TODO(), nil)

	req := &llm.ChatRequest{
		Model:    "qwen3:14b_devstral",
		Messages: conversation,
		Tools:    a.toolDefs,
	}

	resp, err := a.client.Chat(reqCtx, req)
	if err != nil {
		return llm.Message{}, err
	}
	defer resp.Close()

	// This is a simplified example. In a real implementation, you would
	// handle the streaming response and adapt it to the io.ReadCloser interface.
	// For now, we'll just read the entire response.
	body, err := io.ReadAll(resp)
	if err != nil {
		return llm.Message{}, err
	}

	var message llm.Message
	err = json.Unmarshal(body, &message)
	if err != nil {
		return llm.Message{}, err
	}

	return message, nil
}

func dumpConvo(convo []llm.Message) string {
	sb := strings.Builder{}
	for _, m := range convo {
		b, err := json.MarshalIndent(m, "", "    ")
		if err != nil {
			panic(fmt.Sprintf("error unmarshaling %v", m))
		}
		sb.Write(b)
	}
	return sb.String()
}

func ping() error {
	cmd := exec.Command("mpv", "/home/moritz/new-notification-09-352705.mp3")
	err := cmd.Run()
	return err
}