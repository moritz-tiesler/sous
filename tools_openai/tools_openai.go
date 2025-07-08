package toolsopenai

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/openai/openai-go"
)

func ReadFile(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
	path := args["filePath"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func Shell(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
	cmdString := args["command"].(string)

	cmd := exec.Command("bash", "-c", cmdString)
	res, err := cmd.CombinedOutput()
	return string(res), err
}

func WriteFile(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
	path := args["filePath"].(string)
	content := args["content"].(string)

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File edited successfully", nil
}

func SearchFile(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
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

func ListFiles(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
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

func CreateFile(arguments string) (string, error) {
	var args map[string]interface{}
	err := json.Unmarshal([]byte(arguments), &args)
	if err != nil {
		log.Fatalln(err)
	}
	path := args["filePath"].(string)
	content := args["content"].(string)
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "File created successfully", nil
}

func ToolMap() map[string]func(string) (string, error) {
	return map[string]func(string) (string, error){
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

func Tools() []openai.ChatCompletionToolParam {
	tt := []openai.ChatCompletionToolParam{

		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        READ_FILE,
				Description: openai.String("Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names."),
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"filePath"},
					Properties: ToolFunctionProperties{
						"filePath": {
							Type:        "string",
							Description: "The relative path of a file in the working directory.",
						},
					},
				}.ToAPI(),
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        SHELL,
				Description: openai.String("use the shell to execute common linux commands for file manipulation and analysis"),
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"command"},
					Properties: ToolFunctionProperties{
						"command": {
							Type:        "string",
							Description: "the shell command you want to execute",
						},
					},
				}.ToAPI(),
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        WRITE_FILE,
				Description: openai.String("write the contents to a file at a given path. Provides full control over file content. Overwrite existing content. Use with caution."),
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
				}.ToAPI(),
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        SEARCH_FILE,
				Description: openai.String("Search for a string in a file and return matching lines."),
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
				}.ToAPI(),
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        LIST_FILES,
				Description: openai.String("List all files in a directory (or subdirectories, if needed)."),
				Parameters: ToolFunctionParameters{
					Type:     "object",
					Required: []string{"dirPath"},
					Properties: ToolFunctionProperties{
						"dirPath": {
							Type:        "string",
							Description: "the path of the dir to list",
						},
					},
				}.ToAPI(),
			},
		},
		{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        CREATE_FILE,
				Description: openai.String("Create a new file with given content"),
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
				}.ToAPI(),
			},
		},
	}
	return tt
}

type ToolFunctionParameters struct {
	Type       string                 `json:"type"`
	Defs       any                    `json:"$defs,omitempty"`
	Items      any                    `json:"items,omitempty"`
	Required   []string               `json:"required"`
	Properties ToolFunctionProperties `json:"properties"`
}

func (t ToolFunctionParameters) ToAPI() map[string]any {
	d, err := json.Marshal(t)
	if err != nil {
		log.Fatalln(err)
	}
	result := make(map[string]any)
	err = json.Unmarshal(d, &result)
	if err != nil {
		log.Fatalln(err)
	}
	return result
}

type ToolFunctionProperties map[string]ToolFunctionProperty

type ToolFunctionProperty struct {
	Type        string `json:"type"`
	Items       any    `json:"items,omitempty"`
	Description string `json:"description"`
	Enum        []any  `json:"enum,omitempty"`
}

func (t ToolFunctionProperties) ToAPI() map[string]struct {
	Type        string `json:"type"`
	Items       any    `json:"items,omitempty"`
	Description string `json:"description"`
	Enum        []any  `json:"enum,omitempty"`
} {
	result := map[string]struct {
		Type        string `json:"type"`
		Items       any    `json:"items,omitempty"`
		Description string `json:"description"`
		Enum        []any  `json:"enum,omitempty"`
	}{}

	for key, val := range t {
		result[key] = val
	}

	return result
}
