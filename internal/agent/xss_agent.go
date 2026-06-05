package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// XSSAgent - XSS特定测试Agent
type XSSAgent struct {
	llmClient *llm.Client
}

func NewXSSAgent(llmClient *llm.Client) *XSSAgent {
	return &XSSAgent{
		llmClient: llmClient,
	}
}

func (a *XSSAgent) Name() string {
	return "XSSAgent"
}

func (a *XSSAgent) Description() string {
	return "XSS testing agent with predefined workflow"
}

// Execute 执行XSS测试
func (a *XSSAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🎯 [XSSAgent] 开始XSS测试: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))

	// 预定义的XSS测试流程
	workflow := []struct {
		step string
		desc string
	}{
		{"1. 识别输入点", "查找所有可能的用户输入位置"},
		{"2. 测试反射型XSS", "<script>alert(1)</script>"},
		{"3. 测试存储型XSS", "测试能否保存恶意payload"},
		{"4. 测试DOM型XSS", "location.hash, document.write等"},
		{"5. 尝试绕过WAF", "编码、混淆等绕过技巧"},
		{"6. 生成报告", "整理发现的漏洞信息"},
	}

	for i, wf := range workflow {
		fmt.Printf("\n📋 步骤 %d: %s\n", i+1, wf.step)
		fmt.Printf("   描述: %s\n", wf.desc)
	}

	// 模拟执行过程
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成XSS测试报告
func (a *XSSAgent) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                          XSS TEST REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 任务: %s
📅 时间: 当前时间
⚙️  Agent: XSSAgent (特定Agent - 有明确流程)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ 测试发现 (示例):
  1. 发现反射型XSS漏洞 - 中风险
  2. 发现存储型XSS漏洞 - 高风险
  3. 影响范围: 登录页面、留言板、搜索功能
  4. 可窃取Cookie、会话劫持、重定向钓鱼

🛠️ 修复建议:
  - 对用户输入进行输出编码 (HTML实体编码)
  - 使用Content Security Policy (CSP)
  - 设置HttpOnly和Secure标志的Cookie
  - 验证和过滤用户输入

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}
