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

// CodeGenerator - 代码生成Agent
// 负责根据需求生成高质量代码并保存到文件
type CodeGenerator struct {
	gateway   *llm.SimpleGateway
	workDir   string
	fileIndex int
}

// NewCodeGenerator 创建CodeGenerator Agent
func NewCodeGenerator(gateway *llm.SimpleGateway, workDir string) *CodeGenerator {
	return &CodeGenerator{
		gateway:   gateway,
		workDir:   workDir,
		fileIndex: 1,
	}
}

// Name 实现Agent接口
func (a *CodeGenerator) Name() string {
	return "CodeGenerator"
}

// Description 实现Agent接口
func (a *CodeGenerator) Description() string {
	return "Code Generator - Generates clean, efficient code in various programming languages and saves to files"
}

// Execute 执行代码生成
func (a *CodeGenerator) Execute(ctx context.Context, input string) (string, error) {
	// 构建提示词
	codeBlockStart := "```"
	codeBlockEnd := "```"
	
	prompt := "You are a senior software developer. Write clean, efficient code based on the following requirements:\n\n" +
		"Requirements: " + input + "\n\n" +
		"## Output Format\n" +
		"Provide code in markdown code blocks with file extension indicator.\n" +
		"Example:\n" +
		codeBlockStart + "html\n" +
		"<!DOCTYPE html>\n<html>...</html>\n" +
		codeBlockEnd + "\n\n" +
		codeBlockStart + "css\n" +
		"body { ... }\n" +
		codeBlockEnd + "\n\n" +
		codeBlockStart + "javascript\n" +
		"// JavaScript code\n" +
		codeBlockEnd + "\n\n" +
		"Please provide:\n" +
		"1. The complete code implementation\n" +
		"2. Comments explaining key parts\n" +
		"3. File names if applicable\n" +
		"4. Keep code blocks separate for each file\n\n" +
		"Use proper coding conventions and ensure the code is production-ready."

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	savedFiles, err := a.saveCodeBlocks(response)
	if err != nil {
		return fmt.Sprintf("代码生成成功，但保存文件时出错: %v\n\n生成的代码:\n%s", err, response), nil
	}

	result := "代码生成成功！\n\n"
	if len(savedFiles) > 0 {
		result += "已保存的文件:\n"
		for _, file := range savedFiles {
			result += fmt.Sprintf("  - %s\n", file)
		}
		result += "\n"
	}
	result += "生成的代码:\n"
	result += response

	return result, nil
}

func (a *CodeGenerator) saveCodeBlocks(content string) ([]string, error) {
	var savedFiles []string

	processor := messaging.NewMessageProcessor()
	blocks := processor.ExtractCodeBlocks(content)

	for _, block := range blocks {
		fileName := a.generateFileName(block.Language)
		filePath := filepath.Join(a.workDir, fileName)

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return savedFiles, fmt.Errorf("failed to create directory: %v", err)
		}

		if err := os.WriteFile(filePath, []byte(block.Code), 0644); err != nil {
			return savedFiles, fmt.Errorf("failed to write file %s: %v", fileName, err)
		}

		savedFiles = append(savedFiles, fileName)
		a.fileIndex++
	}

	return savedFiles, nil
}

func (a *CodeGenerator) generateFileName(ext string) string {
	nameMap := map[string]string{
		"html":      "index.html",
		"css":       "style.css",
		"js":        "script.js",
		"javascript": "script.js",
		"go":        fmt.Sprintf("main_%d.go", a.fileIndex),
		"python":    fmt.Sprintf("app_%d.py", a.fileIndex),
		"py":        fmt.Sprintf("app_%d.py", a.fileIndex),
		"java":      fmt.Sprintf("Main_%d.java", a.fileIndex),
		"cpp":       fmt.Sprintf("main_%d.cpp", a.fileIndex),
		"c":         fmt.Sprintf("main_%d.c", a.fileIndex),
		"json":      "config.json",
		"yml":       "config.yaml",
		"yaml":      "config.yaml",
		"md":        "README.md",
		"sql":       fmt.Sprintf("schema_%d.sql", a.fileIndex),
	}

	if name, ok := nameMap[strings.ToLower(ext)]; ok {
		return name
	}

	return fmt.Sprintf("code_%d.%s", a.fileIndex, ext)
}