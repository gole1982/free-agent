package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// CommandInjectAgent - 命令注入特定测试Agent
type CommandInjectAgent struct {
	llmClient *llm.Client
}

func NewCommandInjectAgent(llmClient *llm.Client) *CommandInjectAgent {
	return &CommandInjectAgent{llmClient: llmClient}
}

func (a *CommandInjectAgent) Name() string {
	return "CommandInjectAgent"
}

func (a *CommandInjectAgent) Description() string {
	return "Command Injection testing agent with predefined workflow"
}

func (a *CommandInjectAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n💻 [CommandInjectAgent] 开始命令注入测试: %s\n", task)
	return a.generateGenericReport("Command Injection", task), nil
}

// PathTraversalAgent - 路径遍历特定测试Agent
type PathTraversalAgent struct {
	llmClient *llm.Client
}

func NewPathTraversalAgent(llmClient *llm.Client) *PathTraversalAgent {
	return &PathTraversalAgent{llmClient: llmClient}
}

func (a *PathTraversalAgent) Name() string {
	return "PathTraversalAgent"
}

func (a *PathTraversalAgent) Description() string {
	return "Path Traversal testing agent with predefined workflow"
}

func (a *PathTraversalAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n📁 [PathTraversalAgent] 开始路径遍历测试: %s\n", task)
	return a.generateGenericReport("Path Traversal", task), nil
}

// SSRFAgent - SSRF特定测试Agent
type SSRFAgent struct {
	llmClient *llm.Client
}

func NewSSRFAgent(llmClient *llm.Client) *SSRFAgent {
	return &SSRFAgent{llmClient: llmClient}
}

func (a *SSRFAgent) Name() string {
	return "SSRFAgent"
}

func (a *SSRFAgent) Description() string {
	return "SSRF testing agent with predefined workflow"
}

func (a *SSRFAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🌐 [SSRFAgent] 开始SSRF测试: %s\n", task)
	return a.generateGenericReport("SSRF", task), nil
}

// FileIncludeAgent - 文件包含特定测试Agent
type FileIncludeAgent struct {
	llmClient *llm.Client
}

func NewFileIncludeAgent(llmClient *llm.Client) *FileIncludeAgent {
	return &FileIncludeAgent{llmClient: llmClient}
}

func (a *FileIncludeAgent) Name() string {
	return "FileIncludeAgent"
}

func (a *FileIncludeAgent) Description() string {
	return "File Inclusion testing agent with predefined workflow"
}

func (a *FileIncludeAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n📄 [FileIncludeAgent] 开始文件包含测试: %s\n", task)
	return a.generateGenericReport("File Inclusion", task), nil
}

// CTFExploration - CTF探索Agent
type CTFExploration struct {
	llmClient *llm.Client
}

func NewCTFExploration(llmClient *llm.Client) *CTFExploration {
	return &CTFExploration{llmClient: llmClient}
}

func (a *CTFExploration) Name() string {
	return "CTFExploration"
}

func (a *CTFExploration) Description() string {
	return "CTF exploration agent for multi-approach challenges"
}

func (a *CTFExploration) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🏴 [CTFExploration] 开始CTF探索: %s\n", task)
	fmt.Println("   可能需要尝试多种方法...")
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                         CTF EXPLORE RESULT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
⚙️  Agent: CTFExploration (可能需要多方法尝试)

探索完成! (示例占位符)
`, task), nil
}

// 通用报告生成方法
func (a *CommandInjectAgent) generateGenericReport(vulnType, task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    %s TEST REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
⚙️  Agent: %sAgent (特定Agent - 有明确流程)

测试完成! (示例占位符)
`, strings.ToUpper(vulnType), task, vulnType)
}

func (a *PathTraversalAgent) generateGenericReport(vulnType, task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    %s TEST REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
⚙️  Agent: PathTraversalAgent (特定Agent - 有明确流程)

测试完成! (示例占位符)
`, strings.ToUpper(vulnType), task)
}

func (a *SSRFAgent) generateGenericReport(vulnType, task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    %s TEST REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
⚙️  Agent: SSRFAgent (特定Agent - 有明确流程)

测试完成! (示例占位符)
`, strings.ToUpper(vulnType), task)
}

func (a *FileIncludeAgent) generateGenericReport(vulnType, task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    %s TEST REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
⚙️  Agent: FileIncludeAgent (特定Agent - 有明确流程)

测试完成! (示例占位符)
`, strings.ToUpper(vulnType), task)
}
