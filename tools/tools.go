package tools

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  ToolFunctionParameters `json:"parameters"`
}

type ToolFunctionParameters struct {
	Type       string                 `json:"type"`
	Properties ToolFunctionProperties `json:"properties"`
	Required   []string               `json:"required"`
}

type ToolFunctionProperties map[string]ToolFunctionProperty

type ToolFunctionProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func ReadFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func Shell(args map[string]interface{}) (string, error) {
	cmdString := args["command"].(string)
	cmd := exec.Command("bash", "-c", cmdString)
	res, err := cmd.CombinedOutput()
	return string(res), err
}

func WriteFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File edited successfully", nil
}

func SearchFile(args map[string]interface{}) (string, error) {
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

func CreateFile(args map[string]interface{}) (string, error) {
	path := args["filePath"].(string)
	content := args["content"].(string)
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File created successfully", nil
}

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

func Tools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: Function{
				Name:        "readFile",
				Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        "string",
							Description: "The relative path of a file in the working directory.",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "shell",
				Description: "use the shell to execute common linux commands for file manipulation and analysis",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"command"},
					Properties: ToolFunctionProperties{
						"command": {
							Type:        "string",
							Description: "the shell command you want to execute",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "writeFile",
				Description: "write the contents to a file at a given path. Provides full control over file content. Overwrite existing content. Use with caution.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        "string",
							Description: "The relative path of the file in the working directory.",
						},
						"content": {
							Type:        "string",
							Description: "The content to write to the file. All previous content in the file be truncated.",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "searchFile",
				Description: "Search for a string in a file and return matching lines.",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "query"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        "string",
							Description: "The relative path of the file in the working directory.",
						},
						"query": {
							Type:        "string",
							Description: "the string to look for",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "listFiles",
				Description: "List all files in a directory (or subdirectories, if needed).",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"dirPath"},
					Properties: ToolFunctionProperties{
						"dirPath": {
							Type:        "string",
							Description: "the path of the dir to list",
						},
					},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "createFile",
				Description: "Create a new file with given content",
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath", "content"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        "string",
							Description: "the path of the new file",
						},
						"content": {
							Type:        "string",
							Description: "the content of the new file",
						},
					},
				},
			},
		},
	}
}

func (t Tool) ToJSON() string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	return string(b)
}