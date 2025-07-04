package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"slices"
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

	tools := []ToolDefinition{}
	agent := NewAgent(client, getUserMessage, tools)
	err = agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func NewAgent(
	client *api.Client,
	getUserMessage func() (string, bool),
	toosl []ToolDefinition,
) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		toolDefs: []ToolDefinition{
			{
				Name:        "readFile",
				Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
				Function:    ReadFile,
				Params: map[string]string{
					"filePath": "The relative path of a file in the working directory.",
				},
			},
			{
				Name:        "shell",
				Description: "use the shell to execute common linux commands for file manipulation and analysis",
				Function:    Shell,
				Params: map[string]string{
					"command": "the shell command you want to execute",
				},
			},
		},
	}
}

type Agent struct {
	client         *api.Client
	getUserMessage func() (string, bool)
	toolDefs       []ToolDefinition
}

func ReadFile(args api.ToolCallFunctionArguments) (string, error) {
	path := args["filePath"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func Shell(args api.ToolCallFunctionArguments) (string, error) {
	cmdString := args["command"].(string)

	cmd := exec.Command("bash", "-c", cmdString)
	fmt.Println(cmd.Args)
	res, err := cmd.CombinedOutput()
	fmt.Printf("cmd result: %s\n, cmd error: %v", string(res), err)
	return string(res), err
}

const PREFIX = "\u001b[93mDevstral\u001b[0m: %s"

func (a *Agent) Run(ctx context.Context) error {
	conversation := []api.Message{}
	fmt.Println("Chat with Devstral (use 'ctrl-c' to quit)")

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
				res += fmt.Sprintf("error: %s", err.Error())
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
		fmt.Println(dumpConvo(conversation))

	}
	return nil
}

func (a *Agent) executeTool(idx int, name string, args api.ToolCallFunctionArguments) (string, error) {
	var toolDef ToolDefinition
	var found bool
	for _, tool := range a.toolDefs {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}
	if !found {
		return fmt.Sprintf("tool '%s' not found", name), nil
	}
	fmt.Printf("\u001b[92mtool\u001b[0m: %s(%s)\n", name, args)
	response, err := toolDef.Function(args)
	if err != nil {
		fmt.Println(err.Error())
		return err.Error(), err
	}
	return response, nil
}

func (a *Agent) runInference(
	ctx context.Context,
	conversation []api.Message,
	stream bool,
) (api.Message, error) {
	reqTools := api.Tools{}
	for _, td := range a.toolDefs {
		t := api.Tool{
			Type:     "function",
			Function: td.Func(),
		}
		reqTools = append(reqTools, t)
	}

	req := &api.ChatRequest{
		// Model:  "gemma2",
		// Model:    "devstral:24b-small-2505-q8_0",
		// Model:    "llama3.2:latest",
		// Model:    "qwen2.5-coder:32b",
		// Model:    "qwen3:32b",
		Model:    "qwen3:14b",
		Messages: conversation,
		Stream:   &stream,
		Tools:    reqTools,
	}
	content := strings.Builder{}
	message := api.Message{
		Role:      "assistant",
		ToolCalls: []api.ToolCall{},
	}
	respFunc := func(cr api.ChatResponse) error {
		if stream {
			if !cr.Done {
				fmt.Printf("%s", cr.Message.Content)
			} else {
				fmt.Println()
			}
		}
		if len(cr.Message.ToolCalls) > 0 {
			message.ToolCalls = append(message.ToolCalls, cr.Message.ToolCalls...)
		}
		content.WriteString(cr.Message.Content)
		return nil
	}

	fmt.Printf(PREFIX, "")
	err := a.client.Chat(ctx, req, respFunc)
	message.Content = content.String()
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

type ToolDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Schema      api.Tool `json:"schema"`
	Function    func(api.ToolCallFunctionArguments) (string, error)
	Params      map[string]string
}

func (tool ToolDefinition) Func() api.ToolFunction {
	tf := api.ToolFunction{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters: struct {
			Type       string   "json:\"type\""
			Defs       any      "json:\"$defs,omitempty\""
			Items      any      "json:\"items,omitempty\""
			Required   []string "json:\"required\""
			Properties map[string]struct {
				Type        api.PropertyType `json:"type"`
				Items       any              `json:"items,omitempty"`
				Description string           `json:"description"`
				Enum        []any            `json:"enum,omitempty"`
			} `json:"properties"`
		}{
			Type:     "object",
			Required: slices.Collect(maps.Keys(tool.Params)),
			Properties: map[string]struct {
				Type        api.PropertyType "json:\"type\""
				Items       any              "json:\"items,omitempty\""
				Description string           "json:\"description\""
				Enum        []any            "json:\"enum,omitempty\""
			}{},
		},
	}

	for k, v := range tool.Params {
		p := struct {
			Type        api.PropertyType "json:\"type\""
			Items       any              "json:\"items,omitempty\""
			Description string           "json:\"description\""
			Enum        []any            "json:\"enum,omitempty\""
		}{}
		p.Description = v
		p.Type = []string{"string"}
		tf.Parameters.Properties[k] = p
	}
	return tf
}
