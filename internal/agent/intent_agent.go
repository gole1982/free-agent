package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// IntentType 定义支持的意图类型
type IntentType string

const (
	IntentCode        IntentType = "CODE"        // 代码编写
	IntentPlan        IntentType = "PLAN"        // 任务规划
	IntentReview      IntentType = "REVIEW"      // 代码审查
	IntentTest        IntentType = "TEST"        // 测试编写
	IntentDebug       IntentType = "DEBUG"       // 调试分析
	IntentGit         IntentType = "GIT"         // Git操作
	IntentFeedback    IntentType = "FEEDBACK"    // 结果评估
	IntentPentest     IntentType = "PENTEST"     // 安全测试
	IntentChat        IntentType = "CHAT"        // 闲聊/问答
	IntentUnknown     IntentType = "UNKNOWN"     // 未知
)

// IntentResult 意图分析结果
type IntentResult struct {
	Intent     IntentType            `json:"intent"`      // 识别的意图类型
	Confidence float64               `json:"confidence"`  // 置信度 0-1
	AgentName  string                `json:"agent"`       // 应该调用的Agent名称
	Parameters map[string]string     `json:"parameters"`  // 提取的参数
	Summary    string                `json:"summary"`    // 任务摘要
	NeedPlan   bool                  `json:"need_plan"`   // 是否需要先规划
	NeedReview bool                  `json:"need_review"` // 是否需要后续审查
}

// IntentAgent 自然语言理解Agent
type IntentAgent struct {
	gateway *llm.SimpleGateway
}

func NewIntentAgent(gateway *llm.SimpleGateway) *IntentAgent {
	return &IntentAgent{gateway: gateway}
}

func (a *IntentAgent) Name() string {
	return "Intent"
}

func (a *IntentAgent) Description() string {
	return "Natural language understanding. Analyzes user intent and extracts structured parameters."
}

// IntentSystemPrompt 意图识别的系统提示
const IntentSystemPrompt = `
You are a Natural Language Understanding system. Your task is to classify user intent.

## Supported Intents (pick ONE)
- CODE: Creating websites, writing code, implementing features
- PLAN: Planning projects, creating roadmaps
- REVIEW: Code review, quality analysis
- TEST: Writing tests
- DEBUG: Debugging, fixing errors
- GIT: Git operations
- FEEDBACK: Evaluating results
- PENTEST: Security testing
- CHAT: General conversation
- UNKNOWN: Cannot determine

## Rules
1. RESPOND WITH ONLY JSON - no explanations, no markdown, no text before or after
2. Use EXACT field names as shown below
3. confidence must be between 0.0 and 1.0
4. agent must be one of: Coder, Planner, Reviewer, Tester, Debugger, Git, Feedback, Pentesting, Orchestrator

## Output Format
{"intent":"INTENT","confidence":0.95,"agent":"AgentName","summary":"brief summary","need_plan":false,"need_review":true}

## Examples
Input: "创建网页显示天气"
Output: {"intent":"CODE","confidence":0.95,"agent":"Coder","summary":"Create weather display webpage","need_plan":false,"need_review":true}

Input: "规划项目"
Output: {"intent":"PLAN","confidence":0.98,"agent":"Planner","summary":"Plan project","need_plan":false,"need_review":false}
`

func (a *IntentAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("%s\n\n## User Input\n%s", IntentSystemPrompt, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", fmt.Errorf("intent analysis failed: %w", err)
	}

	// 尝试解析JSON
	result, err := a.parseIntentResponse(response)
	if err != nil {
		// 如果JSON解析失败，尝试从文本中推断
		return a.fallbackIntentParse(input)
	}

	return a.formatIntentResult(result)
}

// parseIntentResponse 解析LLM返回的JSON
func (a *IntentAgent) parseIntentResponse(response string) (*IntentResult, error) {
	// 清理响应文本
	jsonStr := cleanResponse(response)
	
	var result IntentResult
	result.Parameters = make(map[string]string)
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse intent JSON: %w", err)
	}

	// 验证intent类型
	if !isValidIntent(result.Intent) {
		result.Intent = IntentUnknown
		result.AgentName = "Orchestrator"
	}

	// 如果agent为空或无效，根据intent填充
	if result.AgentName == "" || !isValidAgentName(result.AgentName) {
		result.AgentName = intentToAgent(result.Intent)
	}

	return &result, nil
}

// fallbackIntentParse 备用解析方法
func (a *IntentAgent) fallbackIntentParse(input string) (string, error) {
	inputLower := strings.ToLower(input)

	// 简单的关键词匹配作为后备
	intent := IntentUnknown
	confidence := 0.5

	keywords := map[string]IntentType{
		"创建": IntentCode, "写": IntentCode, "实现": IntentCode, "网站": IntentCode,
		"网页": IntentCode, "函数": IntentCode, "代码": IntentCode, "html": IntentCode,
		"css": IntentCode, "javascript": IntentCode, "python": IntentCode, "go": IntentCode,
		
		"规划": IntentPlan, "计划": IntentPlan, "分解": IntentPlan, "roadmap": IntentPlan,
		
		"审查": IntentReview, "review": IntentReview, "检查": IntentReview, "分析": IntentReview,
		
		"测试": IntentTest, "test": IntentTest, "用例": IntentTest,
		
		"调试": IntentDebug, "debug": IntentDebug, "错误": IntentDebug, "bug": IntentDebug,
		
		"git": IntentGit, "commit": IntentGit, "push": IntentGit, "pull": IntentGit,
		"branch": IntentGit, "仓库": IntentGit, "版本": IntentGit,
		
		"评估": IntentFeedback, "反馈": IntentFeedback, "改进": IntentFeedback,
		
		"安全": IntentPentest, "渗透": IntentPentest, "ctf": IntentPentest, "漏洞": IntentPentest,
		"sql注入": IntentPentest, "xss": IntentPentest, "注入": IntentPentest,
	}

	for keyword, intentType := range keywords {
		if strings.Contains(inputLower, keyword) {
			intent = intentType
			confidence = 0.7
			break
		}
	}

	result := IntentResult{
		Intent:     intent,
		Confidence: confidence,
		AgentName:  intentToAgent(intent),
		Summary:    input,
		Parameters: make(map[string]string),
		NeedReview: intent == IntentCode,
	}

	return a.formatIntentResult(&result)
}

// formatIntentResult 格式化输出
func (a *IntentAgent) formatIntentResult(result *IntentResult) (string, error) {
	// 返回结构化的结果字符串
	output := fmt.Sprintf(`INTENT_RESULT:
{
  "intent": "%s",
  "confidence": %.2f,
  "agent": "%s",
  "summary": "%s",
  "need_plan": %v,
  "need_review": %v
}`,
		result.Intent,
		result.Confidence,
		result.AgentName,
		result.Summary,
		result.NeedPlan,
		result.NeedReview,
	)

	// 添加参数信息
	if len(result.Parameters) > 0 {
		output += "\nPARAMETERS:"
		for k, v := range result.Parameters {
			output += fmt.Sprintf("\n  %s: %s", k, v)
		}
	}

	return output, nil
}

// isValidIntent 验证意图类型
func isValidIntent(intent IntentType) bool {
	validIntents := []IntentType{
		IntentCode, IntentPlan, IntentReview, IntentTest,
		IntentDebug, IntentGit, IntentFeedback, IntentPentest,
		IntentChat, IntentUnknown,
	}

	for _, v := range validIntents {
		if intent == v {
			return true
		}
	}
	return false
}

// intentToAgent 意图类型到Agent名称的映射
func intentToAgent(intent IntentType) string {
	mapping := map[IntentType]string{
		IntentCode:     "Coder",
		IntentPlan:     "Planner",
		IntentReview:   "Reviewer",
		IntentTest:     "Tester",
		IntentDebug:    "Debugger",
		IntentGit:      "Git",
		IntentFeedback: "Feedback",
		IntentPentest:  "Pentesting",
		IntentChat:     "Orchestrator",
		IntentUnknown:  "Orchestrator",
	}

	if agent, ok := mapping[intent]; ok {
		return agent
	}
	return "Orchestrator"
}

// GetIntentResult 解析并返回结构化的意图结果（供Orchestrator使用）
func GetIntentResult(response string) (*IntentResult, error) {
	// 清理响应文本
	jsonStr := cleanResponse(response)
	
	// 查找JSON对象的边界
	startIdx := strings.Index(jsonStr, "{")
	if startIdx == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}
	
	// 从第一个{开始，找到匹配的}
	depth := 0
	endIdx := -1
	for i := startIdx; i < len(jsonStr); i++ {
		if jsonStr[i] == '{' {
			depth++
		} else if jsonStr[i] == '}' {
			depth--
			if depth == 0 {
				endIdx = i
				break
			}
		}
	}
	
	if endIdx == -1 {
		return nil, fmt.Errorf("invalid JSON structure")
	}
	
	jsonStr = jsonStr[startIdx : endIdx+1]
	
	// 解析JSON
	var result IntentResult
	result.Parameters = make(map[string]string) // 初始化map
	
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// 确保agent名称正确
	if result.AgentName == "" {
		result.AgentName = intentToAgent(result.Intent)
	}

	// 验证agent名称是否有效
	if !isValidAgentName(result.AgentName) {
		result.AgentName = intentToAgent(result.Intent)
	}

	return &result, nil
}

// cleanResponse 清理响应文本，移除多余内容
func cleanResponse(response string) string {
	response = strings.TrimSpace(response)
	
	// 移除markdown代码块
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	
	// 移除可能的解释文本（只保留JSON部分）
	if idx := strings.Index(response, "{"); idx > 0 {
		response = response[idx:]
	}
	
	return response
}

// isValidAgentName 验证agent名称是否有效
func isValidAgentName(name string) bool {
	validAgents := map[string]bool{
		"Coder": true, "Planner": true, "Reviewer": true,
		"Tester": true, "Debugger": true, "Git": true,
		"Feedback": true, "Pentesting": true, "Orchestrator": true,
	}
	return validAgents[name]
}
