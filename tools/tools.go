package tools

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ollama/ollama/api"
)

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

func ToolMap() map[string]func(api.ToolCallFunctionArguments) (string, error) {
	return map[string]func(api.ToolCallFunctionArguments) (string, error){
		"readFile": ReadFile,
		"shell":    Shell,
		"editFile": EditFile,
	}
}

func Tools() api.Tools {
	return api.Tools{
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "readFile",
				Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        api.PropertyType{"string"},
							Description: "The relative path of a file in the working directory.",
						},
					},
				}.ToAPI(),
			},
		},
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "shell",
				Description: "use the shell to execute common linux commands for file manipulation and analysis",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"command"},
					Properties: ToolFunctionProperties{
						"command": {
							Type:        api.PropertyType{"string"},
							Description: "the shell command you want to execute",
						},
					},
				}.ToAPI(),
			},
		},
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "editFile",
				Description: "Edit the contents of a file at a given path. Provides full control over file content.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        api.PropertyType{"string"},
							Description: "The relative path of the file in the working directory.",
						},
						"content": {
							Type:        api.PropertyType{"string"},
							Description: "The new content to write to the file.",
						},
					},
				}.ToAPI(),
			},
		},
	}
}

func EditFile(args api.ToolCallFunctionArguments) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File edited successfully", nil
}

type ToolFunctionParameters struct {
	Type       string                 `json:"type"`
	Defs       any                    `json:"$defs,omitempty"`
	Items      any                    `json:"items,omitempty"`
	Required   []string               `json:"required"`
	Properties ToolFunctionProperties `json:"properties"`
}

func (t ToolFunctionParameters) ToAPI() struct {
	Type       string   `json:"type"`
	Defs       any      `json:"$defs,omitempty"`
	Items      any      `json:"items,omitempty"`
	Required   []string `json:"required"`
	Properties map[string]struct {
		Type        api.PropertyType `json:"type"`
		Items       any              `json:"items,omitempty"`
		Description string           `json:"description"`
		Enum        []any            `json:"enum,omitempty"`
	} `json:"properties"`
} {
	return struct {
		Type       string   `json:"type"`
		Defs       any      `json:"$defs,omitempty"`
		Items      any      `json:"items,omitempty"`
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        api.PropertyType `json:"type"`
			Items       any              `json:"items,omitempty"`
			Description string           `json:"description"`
			Enum        []any            `json:"enum,omitempty"`
		} `json:"properties"`
	}{
		Type:       t.Type,
		Defs:       t.Defs,
		Items:      t.Items,
		Required:   t.Required,
		Properties: t.Properties.ToAPI(),
	}
}

type ToolFunctionProperties map[string]ToolFunctionProperty

type ToolFunctionProperty struct {
	Type        api.PropertyType `json:"type"`
	Items       any              `json:"items,omitempty"`
	Description string           `json:"description"`
	Enum        []any            `json:"enum,omitempty"`
}

func (t ToolFunctionProperties) ToAPI() map[string]struct {
	Type        api.PropertyType `json:"type"`
	Items       any              `json:"items,omitempty"`
	Description string           `json:"description"`
	Enum        []any            `json:"enum,omitempty"`
} {
	result := map[string]struct {
		Type        api.PropertyType `json:"type"`
		Items       any              `json:"items,omitempty"`
		Description string           `json:"description"`
		Enum        []any            `json:"enum,omitempty"`
	}{}

	for key, val := range t {
		result[key] = val
	}

	return result
}
