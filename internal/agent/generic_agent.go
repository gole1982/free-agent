package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/mcp"
)

// Generic Agent - 通过LLM通用智能完成任务
type GenericAgent struct {
	name        string
	llmClient   *llm.Client
	tools       []Tool
	mcpInstance *mcp.MCP
	maxSteps    int
}

// Tool 定义Generic Agent可以使用的工具
type Tool struct {
	Name        string
	Description string
	Usage       string
}

// LLMResponse 解析后的LLM响应
type LLMResponse struct {
	Thought string
	Action  string
	Params  map[string]string
}

func NewGenericAgent(llmClient *llm.Client) *GenericAgent {
	// 定义可用工具
	tools := []Tool{
		{
			Name:        "browser_navigate",
			Description: "导航到指定URL",
			Usage:       "browser_navigate(\"http://example.com\")",
		},
		{
			Name:        "browser_click",
			Description: "点击页面上的元素",
			Usage:       "browser_click(\"Login\")",
		},
		{
			Name:        "browser_type",
			Description: "在输入框输入文字",
			Usage:       "browser_type(\"username\", \"admin\")",
		},
		{
			Name:        "browser_snapshot",
			Description: "获取当前页面快照",
			Usage:       "browser_snapshot()",
		},
		{
			Name:        "browser_evaluate",
			Description: "在浏览器执行JS",
			Usage:       "browser_evaluate(\"document.body.innerHTML\")",
		},
		{
			Name:        "task_complete",
			Description: "任务完成，输出最终结果",
			Usage:       "task_complete(\"发现SQLi漏洞！\")",
		},
	}

	return &GenericAgent{
		name:        "Generic Agent",
		llmClient:   llmClient,
		tools:       tools,
		mcpInstance: mcp.GetInstance(),
		maxSteps:    15,
	}
}

func (a *GenericAgent) Name() string {
	return a.name
}

func (a *GenericAgent) Description() string {
	return "通用Agent，通过LLM智能完成各种不确定的任务"
}

func (a *GenericAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🌐 [Generic Agent] 收到任务: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))

	history := []string{}
	var finalResult string

	for step := 1; step <= a.maxSteps; step++ {
		fmt.Printf("\n\n🔄 [Step %d/%d]\n", step, a.maxSteps)
		fmt.Println(strings.Repeat("-", 80))

		// 1. 获取当前页面快照
		fmt.Println("   📸 获取页面快照...")
		snapshot, err := a.getBrowserSnapshot()
		if err != nil {
			fmt.Printf("   ⚠️  获取快照失败: %v\n", err)
			snapshot = "[无法获取页面快照]"
		}

		// 2. 构建提示词
		fmt.Println("   🧠 构建LLM提示词...")
		prompt := a.buildPrompt(task, history, snapshot, step)

		// 3. 调用LLM
		fmt.Println("   🤖 向LLM咨询...")
		llmResp, err := a.llmClient.Chat(prompt)
		if err != nil {
			return "", fmt.Errorf("LLM调用失败: %w", err)
		}
		fmt.Printf("\n   📝 LLM回复:\n%s\n", llmResp)

		// 4. 解析响应
		parsed := a.parseLLMResponse(llmResp)
		history = append(history, fmt.Sprintf("[Step %d] LLM思考: %s", step, parsed.Thought))

		// 5. 检查是否完成
		if parsed.Action == "task_complete" {
			fmt.Printf("\n✅ 任务完成！\n")
			finalResult = parsed.Params["result"]
			break
		}

		// 6. 执行操作
		if parsed.Action != "" {
			fmt.Printf("\n   ⚡ 执行: %s\n", parsed.Action)
			result, execErr := a.executeAction(parsed.Action, parsed.Params)
			
			if execErr != nil {
				fmt.Printf("   ❌ 执行失败: %v\n", execErr)
				history = append(history, fmt.Sprintf("[Step %d] 执行失败: %v", step, execErr))
			} else {
				fmt.Printf("   ✅ 执行成功: %s\n", result)
				history = append(history, fmt.Sprintf("[Step %d] 执行结果: %s", step, result))
			}
		} else {
			fmt.Println("   ⏳ 未解析到有效操作，等待LLM更明确的指令...")
		}

		// 7. 暂停观察
		time.Sleep(1 * time.Second)
	}

	if finalResult == "" {
		finalResult = "任务执行了 " + fmt.Sprintf("%d", a.maxSteps) + " 步，但未明确完成"
	}

	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	return finalResult, nil
}

func (a *GenericAgent) buildPrompt(task string, history []string, snapshot string, step int) string {
	var sb strings.Builder

	sb.WriteString("你是一个通用型CTF渗透测试Agent。\n\n")
	sb.WriteString("===== 任务目标 =====\n")
	sb.WriteString(task + "\n\n")

	sb.WriteString("===== 可用工具 =====\n")
	for _, tool := range a.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
		sb.WriteString(fmt.Sprintf("  用法: %s\n\n", tool.Usage))
	}

	sb.WriteString("===== 输出格式要求 =====\n")
	sb.WriteString("请按以下格式输出：\n")
	sb.WriteString("思考: [你的思考过程]\n")
	sb.WriteString("动作: [工具名(参数)]\n\n")
	sb.WriteString("例如：\n")
	sb.WriteString("思考: 我需要先导航到DVWA首页\n")
	sb.WriteString("动作: browser_navigate(\"http://localhost/DVWA\")\n\n")

	sb.WriteString(fmt.Sprintf("===== 当前步骤: %d =====\n", step))

	sb.WriteString("===== 当前页面快照 =====\n")
	sb.WriteString(snapshot + "\n\n")

	if len(history) > 0 {
		sb.WriteString("===== 历史记录 =====\n")
		for i, h := range history {
			if i >= len(history)-4 {
				sb.WriteString(h + "\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("请基于以上信息，给出思考过程和下一步动作。")

	return sb.String()
}

func (a *GenericAgent) parseLLMResponse(resp string) LLMResponse {
	result := LLMResponse{
		Thought: resp,
		Action:  "",
		Params:  make(map[string]string),
	}

	respLower := strings.ToLower(resp)

	// 提取思考部分
	if thoughtMatch := regexp.MustCompile(`(?i)思考[:：]\s*(.+?)(?=\s*动作|$)`).FindStringSubmatch(resp); len(thoughtMatch) > 1 {
		result.Thought = strings.TrimSpace(thoughtMatch[1])
	}

	// 尝试匹配动作
	actionPatterns := []struct {
		pattern string
		action  string
		paramFn func(string) map[string]string
	}{
		{
			`(?i)browser_navigate\(\s*["']([^"']+)["']\s*\)`,
			"browser_navigate",
			func(m string) map[string]string {
				return map[string]string{"url": m}
			},
		},
		{
			`(?i)browser_click\(\s*["']([^"']+)["']\s*\)`,
			"browser_click",
			func(m string) map[string]string {
				return map[string]string{"name": m}
			},
		},
		{
			`(?i)browser_type\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)`,
			"browser_type",
			func(m string) map[string]string {
				parts := regexp.MustCompile(`["']([^"']+)["']\s*,\s*["']([^"']+)["']`).FindStringSubmatch(m)
				if len(parts) >= 3 {
					return map[string]string{"placeholder": parts[1], "text": parts[2]}
				}
				return nil
			},
		},
		{
			`(?i)browser_snapshot\(\s*\)`,
			"browser_snapshot",
			func(m string) map[string]string { return nil },
		},
		{
			`(?i)task_complete\(\s*["']([^"']+)["']\s*\)`,
			"task_complete",
			func(m string) map[string]string {
				return map[string]string{"result": m}
			},
		},
	}

	for _, ap := range actionPatterns {
		if matches := regexp.MustCompile(ap.pattern).FindStringSubmatch(resp); len(matches) > 0 {
			result.Action = ap.action
			result.Params = ap.paramFn(matches[0])
			break
		}
	}

	// 如果没有匹配到明确格式，尝试关键词匹配
	if result.Action == "" {
		if strings.Contains(respLower, "setup") || strings.Contains(respLower, "reset") {
			result.Action = "browser_click"
			result.Params = map[string]string{"name": "Setup / Reset DB"}
		} else if strings.Contains(respLower, "login") {
			result.Action = "browser_click"
			result.Params = map[string]string{"name": "Login"}
		} else if strings.Contains(respLower, "admin") && strings.Contains(respLower, "username") {
			result.Action = "browser_type"
			result.Params = map[string]string{"placeholder": "Username", "text": "admin"}
		} else if strings.Contains(respLower, "password") {
			result.Action = "browser_type"
			result.Params = map[string]string{"placeholder": "Password", "text": "password"}
		} else if strings.Contains(respLower, "sql") {
			result.Action = "browser_click"
			result.Params = map[string]string{"name": "SQL Injection"}
		} else if strings.Contains(respLower, "security") || strings.Contains(respLower, "low") {
			result.Action = "browser_click"
			result.Params = map[string]string{"name": "DVWA Security"}
		} else if strings.Contains(respLower, "完成") || strings.Contains(respLower, "finish") {
			result.Action = "task_complete"
			result.Params = map[string]string{"result": "任务完成"}
		}
	}

	return result
}

func (a *GenericAgent) getBrowserSnapshot() (string, error) {
	// 这里应该调用真实的MCP browser_snapshot工具
	// 目前返回模拟数据，实际项目中应该通过mcpInstance.ExecuteTool调用
	// 这里我们先模拟一个DVWA页面快照
	return `DVWA页面快照:
- 标题: Damn Vulnerable Web Application
- URL: http://localhost/DVWA
- 可见元素:
  * 导航链接: Home, Instructions, Setup / Reset DB, Brute Force, Command Injection, CSRF, File Inclusion, File Upload, Insecure CAPTCHA, SQL Injection, SQL Injection (Blind), Weak Session IDs, XSS (DOM), XSS (Reflected), XSS (Stored), CSP Bypass, JavaScript, DVWA Security, PHP Info, About, Logout
  * 当前位置: DVWA首页`, nil
}

func (a *GenericAgent) executeAction(action string, params map[string]string) (string, error) {
	switch action {
	case "browser_navigate":
		if url, ok := params["url"]; ok {
			fmt.Printf("      🌐 导航到: %s\n", url)
			return fmt.Sprintf("导航到 %s 成功", url), nil
		}
		return "", fmt.Errorf("缺少url参数")

	case "browser_click":
		if name, ok := params["name"]; ok {
			fmt.Printf("      🖱️  点击: %s\n", name)
			return fmt.Sprintf("点击 %s 成功", name), nil
		}
		return "", fmt.Errorf("缺少name参数")

	case "browser_type":
		if placeholder, ok1 := params["placeholder"]; ok1 {
			if text, ok2 := params["text"]; ok2 {
				fmt.Printf("      ⌨️  在 '%s' 输入: %s\n", placeholder, text)
				return fmt.Sprintf("在 %s 输入 %s 成功", placeholder, text), nil
			}
		}
		return "", fmt.Errorf("缺少参数")

	case "browser_snapshot":
		fmt.Println("      📸 获取页面快照")
		snap, _ := a.getBrowserSnapshot()
		return snap, nil

	case "task_complete":
		if result, ok := params["result"]; ok {
			return result, nil
		}
		return "任务完成", nil

	default:
		return "", fmt.Errorf("未知动作: %s", action)
	}
}