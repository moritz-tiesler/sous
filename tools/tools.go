package tools

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

// Tool defines the structure for a tool that can be called by the agent.
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function defines the structure of a function that can be called by a tool.
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolFunctionParameters defines the structure of the parameters for a tool function.
type ToolFunctionParameters struct {
	Type       string                 `json:"type"`
	Properties ToolFunctionProperties `json:"properties"`
	Required   []string               `json:"required"`
}

// ToolFunctionProperties is a map of property names to their definitions.
type ToolFunctionProperties map[string]ToolFunctionProperty

// ToolFunctionProperty defines a single property for a tool function.
type ToolFunctionProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ReadFile reads the content of a file at the given path.
func ReadFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// Shell executes a shell command.
func Shell(args map[string]interface{}) (string, error) {
	cmdString := args["command"].(string)
	cmd := exec.Command("bash", "-c", cmdString)
	res, err := cmd.CombinedOutput()
	return string(res), err
}

// WriteFile writes content to a file at the given path.
func WriteFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File edited successfully", nil
}

// SearchFile searches for a query string within a file.
func SearchFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	query := args["query"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, query) {
			matches = append(matches, line)
		}
	}
	return strings.Join(matches, "\n"), nil
}

// ListFiles lists the files in a directory.
func ListFiles(args map[string]interface{}) (string, error) {
	dirPath := args["dirPath"].(string)
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	var result []string
	for _, file := range files {
		fName := file.Name()
		if file.IsDir() {
			fName += "/"
		}
		result = append(result, fName)
	}
	return strings.Join(result, "\n"), nil
}

// CreateFile creates a new file with the given content.
func CreateFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File created successfully", nil
}

// ToolMap returns a map of tool names to their implementation.
func ToolMap() map[string]func(map[string]interface{}) (string, error) {
	return map[string]func(map[string]interface{}) (string, error){
		"readFile":   ReadFile,
		"shell":      Shell,
		"writeFile":  WriteFile,
		"searchFile": SearchFile,
		"listFiles":  ListFiles,
		"createFile": CreateFile,
	}
}

// mustMarshal is a helper to marshal JSON without handling the error.
func mustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// Tools returns the list of available tools.
func Tools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "readFile",
				Description: "Read the contents of a given relative file path.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath"},
					Properties: ToolFunctionProperties{
						"filePath": {Type: "string", Description: "The relative path of a file in the working directory."},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "shell",
				Description: "Execute a shell command.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"command"},
					Properties: ToolFunctionProperties{
						"command": {Type: "string", Description: "The shell command to execute."},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "writeFile",
				Description: "Write content to a file.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {Type: "string", Description: "The relative path of the file."},
						"content":  {Type: "string", Description: "The content to write."},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "searchFile",
				Description: "Search for a string in a file.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "query"},
					Properties: ToolFunctionProperties{
						"filePath": {Type: "string", Description: "The relative path of the file."},
						"query":    {Type: "string", Description: "The string to search for."},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "listFiles",
				Description: "List files in a directory.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"dirPath"},
					Properties: ToolFunctionProperties{
						"dirPath": {Type: "string", Description: "The path of the directory to list."},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "createFile",
				Description: "Create a new file with given content.",
				Parameters:  ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {Type: "string", Description: "The path of the new file."},
						"content":  {Type: "string", Description: "The content of the new file."},
					},
				},
			},
		},
	}
}