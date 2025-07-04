package tools

import (
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
	res, err := cmd.CombinedOutput()
	return string(res), err
}

func WriteFile(args api.ToolCallFunctionArguments) (string, error) {
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
		READ_FILE:   ReadFile,
		SHELL:       Shell,
		WRITE_FILE:  WriteFile,
		SEARCH_FILE: SearchFile,
		LIST_FILES:  ListFiles,
		CREATE_FILE: CreateFile,
	}
}

const (
	READ_FILE   = "readFile"
	SHELL       = "shell"
	WRITE_FILE  = "writeFile"
	SEARCH_FILE = "searchFile"
	LIST_FILES  = "listFiles"
	CREATE_FILE = "createFile"
)

func Tools() api.Tools {
	return api.Tools{
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        READ_FILE,
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
				Name:        SHELL,
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
				Name:        WRITE_FILE,
				Description: "write the contents to a file at a given path. Provides full control over file content.",
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
							Description: "The content to write to the file.",
						},
					},
				}.ToAPI(),
			},
		},
		api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        SEARCH_FILE,
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
				Name:        LIST_FILES,
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
				Name:        CREATE_FILE,
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
