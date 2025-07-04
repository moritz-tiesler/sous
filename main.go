package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/moritz-tiesler/sous/tools"
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	appCtx, appCancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-sigCh:
				if agent.chatContext.cancel != nil {
					agent.chatContext.cancel()
					fmt.Println()
					agent.SetActiveChatContext(context.TODO(), nil)
				} else {
					appCancel()
				}
			case <-appCtx.Done():
				// Context was already cancelled from another source (e.g., main exited)
				return
			}
		}
	}()

	go func() {
		err = agent.Run(appCtx)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
	}()

	<-appCtx.Done()
	fmt.Println("Bye")
	os.Exit(1)
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

func (a *Agent) SetActiveChatContext(ctx context.Context, cancel context.CancelFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chatContext = &ChatContext{ctx: ctx, cancel: cancel}
}

const PREFIX = "\u001b[93mSous\u001b[0m: %s"

func (a *Agent) Run(ctx context.Context) error {
	conversation := []api.Message{}
	fmt.Println("Chat with Sous")

	stream := true
	readUserInput := true
	for {
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
			// fmt.Println(dumpConvo(conversation))
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
		// fmt.Println(dumpConvo(conversation))

	}
	return nil
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
	PrintAction("tool: %s, %v\n", name, args)
	response, err := toolFunc(args)
	if err != nil {
		PrintAction("errors %s %v\n", response, err.Error())
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
		// Model:  "gemma2",
		// Model:    "devstral:24b-small-2505-q8_0",
		// Model:    "llama3.2:latest",
		// Model:    "qwen2.5-coder:32b",
		// Model:    "qwen3:32b",
		Model:    "qwen3:14b",
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
			printFunc = PrintThink
		} else {
			printFunc = PrintNonThink
		}
		if !cr.Done {
			printFunc("%s", cr.Message.Content)
		} else {
			fmt.Println()
		}
		if len(cr.Message.ToolCalls) > 0 {
			message.ToolCalls = append(message.ToolCalls, cr.Message.ToolCalls...)
		}
		if !thinkingOutput {
			content.WriteString(cr.Message.Content)
		}
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

func dumpRequest(r api.ChatRequest) string {
	sb := strings.Builder{}
	b, err := json.MarshalIndent(r, "", "    ")
	if err != nil {
		panic(fmt.Sprintf("error unmarshaling %v", r))
	}
	sb.Write(b)
	return sb.String()
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
