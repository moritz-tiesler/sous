package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	fmt.Printf("cmd result: %s\n, cmd error: %v\n", string(res), err)
	return string(res), err
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

func SearchFile(args api.ToolCallFunctionArguments) (string, error) {
	path := args["filePath"].(string)
	query := args["query"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(content), "\n")
	var matches []string
	for _, line := range lines {
		if strings.Contains(line, query) {
			matches = append(matches, line)
		}
	}
	return strings.Join(matches, "\n"), nil
}

func ListFiles(args api.ToolCallFunctionArguments) (string, error) {
	dirPath := args["dirPath"].(string)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	var result []string
	for _, file := range files {
		result = append(result, file.Name())
	}
	return strings.Join(result, "\n"), nil
}

func CreateFile(args api.ToolCallFunctionArguments) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File created successfully", nil
}

func ToolMap() map[string]func(api.ToolCallFunctionArguments) (string, error) {
	return map[string]func(api.ToolCallFunctionArguments) (string, error){
		"readFile":   ReadFile,
		"shell":      Shell,
		"editFile":   EditFile,
		"searchFile": SearchFile,
		"listFiles":  ListFiles,
		"createFile": CreateFile,
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
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "searchFile",
				Description: "Search for a string in a file and return matching lines.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "query"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        api.PropertyType{"string"},
							Description: "The relative path of the file in the working directory.",
						},
						"query": {
							Type:        api.PropertyType{"string"},
							Description: "the string to look for",
						},
					},
				}.ToAPI(),
			},
		},
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "listFiles",
				Description: "List all files in a directory (or subdirectories, if needed).",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"dirPath"},
					Properties: ToolFunctionProperties{
						"dirPath": {
							Type:        api.PropertyType{"string"},
							Description: "the path of the dir to list",
						},
					},
				}.ToAPI(),
			},
		},
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "createFile",
				Description: "Create a new file with given content",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        api.PropertyType{"string"},
							Description: "the path of the new file",
						},
						"content": {
							Type:        api.PropertyType{"string"},
							Description: "the content of the new file",
						},
					},
				}.ToAPI(),
			},
		},
	}
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
