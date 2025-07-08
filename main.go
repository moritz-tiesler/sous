package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/glamour"
	"github.com/moritz-tiesler/sous/client"
	toolsopenai "github.com/moritz-tiesler/sous/tools_openai"
	"github.com/ollama/ollama/api"
	"github.com/openai/openai-go"
)

func main() {
	// client, err := api.ClientFromEnvironment()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	client := client.New("Qwen3-14B-128K-GGUF_Qwen3-14B-128K-UD-Q6_K_XL")

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
				if agent.client.ChatContext.Cancel != nil {
					agent.client.ChatContext.Cancel()
					fmt.Println()
					agent.client.ClearChatContext()
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
		err := agent.Run(appCtx)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			appCancel()
		}
	}()

	<-appCtx.Done()
	fmt.Println("Bye")
	os.Exit(1)
}

func NewAgent(
	client *client.Client,
	getUserMessage func() (string, bool),
) *Agent {
	return &Agent{
		client:       client,
		getUserInput: getUserMessage,
		toolDefs:     toolsopenai.Tools(),
		toolMap:      toolsopenai.ToolMap(),
	}
}

type ChatContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Agent struct {
	client       *client.Client
	getUserInput func() (string, bool)
	toolDefs     []openai.ChatCompletionToolParam
	toolMap      map[string]func(string) (string, error)
}

const PREFIX = "\u001b[93mSous\u001b[0m: %s"

func dumpConvo(convo []openai.ChatCompletionMessageParamUnion) string {
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

func (a *Agent) Run(ctx context.Context) error {
	conversation := []openai.ChatCompletionMessageParamUnion{}
	fmt.Println("Chat with Sous")

	// stream := true
	readUserInput := true
	for {
		if len(conversation) > 4 {
			PrintAction("%s...\n", "SUMMARIZING")
			summary, err := a.summarizeConvo(ctx, conversation)

			if err != nil {
				fmt.Println("error after summarizeConvo")
				fmt.Println(dumpConvo(conversation))
				fmt.Println(err.Error())
			} else {
				conversation = append([]openai.ChatCompletionMessageParamUnion{}, summary.ToParam())
				PrintAction("NEW CONVO LEN=%d...\n", len(conversation))
				PrintAction("NEW CONVO STarts with=%s...\n", summary.Content)
			}
		}
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserInput()
			if !ok {
				break
			}
			userMessage := openai.UserMessage(userInput)
			conversation = append(conversation, userMessage)
		}

		message, err := a.client.RunInference(ctx, conversation)
		if err != nil {
			fmt.Println("error after RunInference")
			fmt.Println(len(conversation))
			fmt.Println(dumpConvo(conversation))
			fmt.Println(err.Error())
		}
		conversation = append(conversation, message.ToParam())

		toolResults := []openai.ChatCompletionMessageParamUnion{}

		fmt.Printf(PREFIX, "")
		// TODO print code md snippets as md
		// PrintNonThink("%s\n", message.Content)
		out, err := glamour.Render(message.Content, "dracula")
		if err != nil {
			panic(err)
		}
		fmt.Print(out)
		for _, toolCall := range message.ToolCalls {
			f := toolCall.Function
			result, _ := a.executeTool(toolCall.ID, f.Name, f.Arguments)
			toolResults = append(toolResults, result)
		}
		if len(toolResults) == 0 {
			readUserInput = true
			go func() {
				if err := ping(); err != nil {
					panic(err.Error())
				}
			}()
			continue
		}

		readUserInput = false
		conversation = append(conversation, toolResults...)

		if len(conversation) < 1 {
			panic("why conve len=0?????")
		}

	}
	return nil
}

func (a *Agent) summarizeConvo(
	ctx context.Context,
	conversation []openai.ChatCompletionMessageParamUnion,
) (openai.ChatCompletionMessage, error) {

	userMessage := openai.UserMessage(
		"please summarize the active conversation, so that you can pick up your work form here. include the original user instructions so that you do not loose the context of the task at hand. include previous tool calls in this summary.",
	)
	conversation = append(conversation, userMessage)

	summary, err := a.client.RunInference(ctx, conversation)
	return summary, err
}

func (a *Agent) executeTool(id string, name string, args string) (openai.ChatCompletionMessageParamUnion, error) {
	var toolFunc func(string) (string, error)
	var found bool
	for n, f := range a.toolMap {
		if n == name {
			toolFunc = f
			found = true
			break
		}
	}
	if !found {
		return openai.ToolMessage("tool '%s' not found", id), nil
	}
	response, err := toolFunc(args)
	PrintAction("tool: %s, %v\n%v\n", name, args, response)
	if err != nil {
		PrintAction("errors %s %v\n", response, err.Error())
		return openai.ToolMessage(err.Error(), id), nil
	}
	return openai.ToolMessage(response, id), nil
}

func (a *Agent) runInference(
	ctx context.Context,
	conversation []openai.ChatCompletionMessageParamUnion,
	// stream bool,
) (openai.ChatCompletionMessage, error) {

	reqCtx, reqCancel := context.WithCancel(ctx)
	a.client.SetActiveChatContext(reqCtx, reqCancel)
	// req := &api.ChatRequest{
	// 	// Model:  "gemma2",
	// 	// Model:    "devstral:24b-small-2505-q8_0",
	// 	// Model:    "llama3.2:latest",
	// 	// Model:    "qwen2.5-coder:32b",
	// 	// Model:    "qwen3:32b",
	// 	// Model:    "qwen3:14b",
	// 	Model:    "qwen3:14b_devstral",
	// 	Messages: conversation,
	// 	Stream:   &stream,
	// 	Tools:    a.toolDefs,
	// }
	// content := strings.Builder{}
	// message := api.Message{
	// 	Role:      "assistant",
	// 	ToolCalls: []api.ToolCall{},
	// }
	// thinkingOutput := false
	// respFunc := func(cr api.ChatResponse) error {
	// 	select {
	// 	case <-reqCtx.Done():
	// 		return fmt.Errorf("chat cancelled")
	// 	default:
	// 	}

	// 	if strings.TrimSpace(cr.Message.Content) == "<think>" {
	// 		thinkingOutput = true
	// 	}
	// 	var printFunc func(format string, a ...interface{})
	// 	if thinkingOutput {
	// 		printFunc = PrintThink
	// 	} else {
	// 		printFunc = PrintNonThink
	// 	}
	// 	if !cr.Done {
	// 		printFunc("%s", cr.Message.Content)
	// 	} else {
	// 		fmt.Println()
	// 	}
	// 	if len(cr.Message.ToolCalls) > 0 {
	// 		message.ToolCalls = append(message.ToolCalls, cr.Message.ToolCalls...)
	// 	}
	// 	content.WriteString(cr.Message.Content)
	// 	if strings.TrimSpace(cr.Message.Content) == "</think>" {
	// 		thinkingOutput = false
	// 	}
	// 	return nil
	// }

	// fmt.Printf(PREFIX, "")
	// err := a.client.Chat(reqCtx, req, respFunc)
	// message.Content = content.String()
	message, _ := a.client.RunInference(ctx, conversation)
	a.client.ClearChatContext()
	return message, nil //, err
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

func ping() error {
	cmd := exec.Command("mpv", "/home/moritz/new-notification-09-352705.mp3")
	err := cmd.Run()
	return err
}
