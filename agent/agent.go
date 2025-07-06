package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/moritz-tiesler/sous/output"
	"github.com/moritz-tiesler/sous/tools"
	"github.com/ollama/ollama/api"
)

type ChatContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Agent struct {
	client         *api.Client
	getUserMessage func() (string, bool)
	toolDefs       api.Tools
	toolMap        map[string]func(api.ToolCallFunctionArguments) (string, error)

	mu          sync.Mutex
	chatContext *ChatContext
}

func NewAgent(
	client *api.Client,
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
	conversation := []api.Message{}
	fmt.Println("Chat with Sous")

	stream := true
	readUserInput := true
	for {
		if len(conversation) > 10 {
			output.PrintAction("%s...\n", "SUMMARIZING")
			summary, err := a.summarizeConvo(ctx, conversation)
			if err != nil {
				return err
			}

			conversation = append([]api.Message{}, summary)
			output.PrintAction("NEW CONVO LEN=%d...\n", len(conversation))
			output.PrintAction("NEW CONVO STarts with=%s...\n", summary.Content)
		}
		if readUserInput {
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
		}

		message, err := a.runInference(ctx, conversation, stream)
		if err != nil {
			fmt.Println(dumpConvo(conversation))
			return err
		}
		conversation = append(conversation, message)

		toolResults := map[string]string{}
		for _, toolCall := range message.ToolCalls {
			f := toolCall.Function
			result, err := a.executeTool(f.Index, f.Name, f.Arguments)
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
		conversation = append(conversation, api.Message{Role: "tool", Content: toolResMessage})

	}
	return nil
}

func (a *Agent) summarizeConvo(ctx context.Context, conversation []api.Message) (api.Message, error) {

	reqCtx, reqCancel := context.WithCancel(ctx)
	a.SetActiveChatContext(reqCtx, reqCancel)
	conversation = append(conversation, api.Message{
		Role:    "user",
		Content: "please summarize the active conversation, so that you can pick up your work form here. include the original user instructions so that you do not loose the context of the task at hand. include previous tool calls in this summary.",
	})
	stream := true
	req := &api.ChatRequest{
		Model:    "qwen3:14b_devstral",
		Messages: conversation,
		Stream:   &stream,
		Tools:    a.toolDefs,
	}
	content := strings.Builder{}
	message := api.Message{
		Role: "user",
	}
	respFunc := func(cr api.ChatResponse) error {
		select {
		case <-reqCtx.Done():
			return fmt.Errorf("chat cancelled")
		default:
		}

		if !cr.Done {
			output.PrintAction("%s", cr.Message.Content)
		} else {
			fmt.Println()
		}
		content.WriteString(cr.Message.Content)
		return nil
	}

	err := a.client.Chat(reqCtx, req, respFunc)
	message.Content = content.String()
	a.SetActiveChatContext(context.TODO(), nil)
	return message, err
}

func (a *Agent) executeTool(idx int, name string, args api.ToolCallFunctionArguments) (string, error) {
	var toolFunc func(api.ToolCallFunctionArguments) (string, error)
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
	response, err := toolFunc(args)
	output.PrintAction("tool: %s, %v\n%v\n", name, args, response)
	if err != nil {
		output.PrintAction("errors %s %v\n", response, err.Error())
		return response, err
	}
	return response, nil
}

func (a *Agent) runInference(
	ctx context.Context,
	conversation []api.Message,
	stream bool,
) (api.Message, error) {

	reqCtx, reqCancel := context.WithCancel(ctx)
	a.SetActiveChatContext(reqCtx, reqCancel)
	req := &api.ChatRequest{
		Model:    "qwen3:14b_devstral",
		Messages: conversation,
		Stream:   &stream,
		Tools:    a.toolDefs,
	}
	content := strings.Builder{}
	message := api.Message{
		Role:      "assistant",
		ToolCalls: []api.ToolCall{},
	}
	thinkingOutput := false
	respFunc := func(cr api.ChatResponse) error {
		select {
		case <-reqCtx.Done():
			return fmt.Errorf("chat cancelled")
		default:
		}

		if strings.TrimSpace(cr.Message.Content) == "<think>" {
			thinkingOutput = true
		}
		var printFunc func(format string, a ...interface{})
		if thinkingOutput {
			printFunc = output.PrintThink
		} else {
			printFunc = output.PrintNonThink
		}
		if !cr.Done {
			printFunc("%s", cr.Message.Content)
		} else {
			fmt.Println()
		}
		if len(cr.Message.ToolCalls) > 0 {
			message.ToolCalls = append(message.ToolCalls, cr.Message.ToolCalls...)
		}
		content.WriteString(cr.Message.Content)
		if strings.TrimSpace(cr.Message.Content) == "</think>" {
			thinkingOutput = false
		}
		return nil
	}

	fmt.Printf(PREFIX, "")
	err := a.client.Chat(reqCtx, req, respFunc)
	message.Content = content.String()
	a.SetActiveChatContext(context.TODO(), nil)
	return message, err
}

func dumpConvo(convo []api.Message) string {
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
