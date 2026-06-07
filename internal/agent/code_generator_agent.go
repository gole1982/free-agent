package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/messaging"
)

type CodeGeneratorAgent struct {
	gateway   *llm.SimpleGateway
	workDir   string
	fileIndex int
}

func NewCodeGeneratorAgent(gateway *llm.SimpleGateway, workDir string) *CodeGeneratorAgent {
	return &CodeGeneratorAgent{
		gateway:   gateway,
		workDir:   workDir,
		fileIndex: 1,
	}
}

func (a *CodeGeneratorAgent) Name() string {
	return "CodeGenerator"
}

func (a *CodeGeneratorAgent) Description() string {
	return "Generates clean, efficient code in various programming languages and writes files to the work directory."
}

func (a *CodeGeneratorAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := "You are a senior software developer. Write clean, efficient code based on the following requirements:\n\n" +
		"Requirements: " + input + "\n\n" +
		"## Output Format\n" +
		"Provide code in markdown code blocks with language tags.\n" +
		"Use proper coding conventions and ensure the code is production-ready."

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	savedFiles, err := a.saveCodeBlocks(response)
	if err != nil {
		return fmt.Sprintf("Code generated but saving failed: %v\n\nOutput:\n%s", err, response), nil
	}

	result := "Code generated successfully.\n\n"
	if len(savedFiles) > 0 {
		result += "Saved files:\n"
		for _, file := range savedFiles {
			result += fmt.Sprintf("  - %s\n", file)
		}
		result += "\n"
	}
	result += "Output:\n" + response
	return result, nil
}

func (a *CodeGeneratorAgent) saveCodeBlocks(content string) ([]string, error) {
	var savedFiles []string
	processor := messaging.NewMessageProcessor()
	blocks := processor.ExtractCodeBlocks(content)

	for _, block := range blocks {
		fileName := a.generateFileName(block.Language)
		filePath := filepath.Join(a.workDir, fileName)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return savedFiles, fmt.Errorf("create dir: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(block.Code), 0644); err != nil {
			return savedFiles, fmt.Errorf("write %s: %v", fileName, err)
		}
		savedFiles = append(savedFiles, fileName)
		a.fileIndex++
	}
	return savedFiles, nil
}

func (a *CodeGeneratorAgent) generateFileName(ext string) string {
	nameMap := map[string]string{
		"html":       "index.html",
		"css":        "style.css",
		"js":         "script.js",
		"javascript": "script.js",
		"go":         fmt.Sprintf("main_%d.go", a.fileIndex),
		"python":     fmt.Sprintf("app_%d.py", a.fileIndex),
		"py":         fmt.Sprintf("app_%d.py", a.fileIndex),
		"java":       fmt.Sprintf("Main_%d.java", a.fileIndex),
		"cpp":        fmt.Sprintf("main_%d.cpp", a.fileIndex),
		"c":          fmt.Sprintf("main_%d.c", a.fileIndex),
		"json":       "config.json",
		"yml":        "config.yaml",
		"yaml":       "config.yaml",
		"md":         "README.md",
		"sql":        fmt.Sprintf("schema_%d.sql", a.fileIndex),
	}
	if name, ok := nameMap[strings.ToLower(ext)]; ok {
		return name
	}
	return fmt.Sprintf("code_%d.%s", a.fileIndex, ext)
}
