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
	// 安全测试类 - 细分
	IntentSQLi        IntentType = "SQLI"        // SQL注入
	IntentXSS         IntentType = "XSS"         // XSS
	IntentCommandInj  IntentType = "CMDINJ"      // 命令注入
	IntentPathTravers IntentType = "PATHTRAV"    // 路径遍历
	IntentSSRF        IntentType = "SSRF"        // SSRF
	IntentFileIncl    IntentType = "FILEINCL"    // 文件包含
	IntentPentest     IntentType = "PENTEST"     // 综合安全测试
	IntentCTF         IntentType = "CTF"         // CTF探索
	IntentChat        IntentType = "CHAT"        // 闲聊/问答
	IntentProject     IntentType = "PROJECT"     // 复杂项目（需要多Agent协作）
	IntentExploration IntentType = "EXPLORATION" // 探索式任务（不确定、需要回溯）
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
- CODE: Creating websites, writing code, implementing features (simple tasks)
- PLAN: Planning projects, creating roadmaps
- REVIEW: Code review, quality analysis
- TEST: Writing tests
- DEBUG: Debugging, fixing errors
- GIT: Git operations
- FEEDBACK: Evaluating results
- SQLI: SQL Injection testing (e.g., "test SQL injection", "SQLi", "' OR 1=1")
- XSS: XSS testing (e.g., "test XSS", "<script>", "javascript:")
- CMDINJ: Command Injection testing (e.g., "command injection", "RCE", "; ls")
- PATHTRAV: Path Traversal testing (e.g., "../", "path traversal", "directory traversal")
- SSRF: SSRF testing (e.g., "SSRF", "http://127.0.0.1")
- FILEINCL: File Inclusion testing (e.g., "LFI", "RFI", "file inclusion")
- PENTEST: Comprehensive security testing (not specific to one vulnerability type)
- CTF: CTF challenges that may require multiple approaches
- CHAT: General conversation
- PROJECT: Complex projects requiring multi-agent collaboration (e.g., "build a complete e-commerce website", "create a full-stack application")
- EXPLORATION: Uncertain problems requiring exploration, backtracking, multi-branch trial
- UNKNOWN: Cannot determine

## Rules
1. RESPOND WITH ONLY JSON - no explanations, no markdown, no text before or after
2. Use EXACT field names as shown below
3. confidence must be between 0.0 and 1.0
4. agent must be one of: Coder, Planner, Reviewer, Tester, Debugger, Git, Feedback, SQLiAgent, XSSAgent, CommandInjectAgent, PathTraversalAgent, SSRFAgent, FileIncludeAgent, Pentesting, CTFExploration, Generic, Orchestrator

## Output Format
{"intent":"INTENT","confidence":0.95,"agent":"AgentName","summary":"brief summary","need_plan":false,"need_review":true}

## Examples
Input: "创建网页显示天气"
Output: {"intent":"CODE","confidence":0.95,"agent":"Coder","summary":"Create weather display webpage","need_plan":false,"need_review":true}

Input: "测试DVWA的SQL注入"
Output: {"intent":"SQLI","confidence":0.98,"agent":"SQLiAgent","summary":"Test SQL injection on DVWA","need_plan":false,"need_review":false}

Input: "测试XSS漏洞"
Output: {"intent":"XSS","confidence":0.95,"agent":"XSSAgent","summary":"Test XSS vulnerability","need_plan":false,"need_review":false}

Input: "做一个CTF挑战"
Output: {"intent":"CTF","confidence":0.85,"agent":"CTFExploration","summary":"Solve CTF challenge","need_plan":false,"need_review":false}

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

	// 优先匹配更具体的关键词（细分的安全测试）
	specificKeywords := []struct {
		keywords []string
		intent   IntentType
		conf     float64
	}{
		// 安全测试 - 细分
		{[]string{"sql注入", "sql injection", "sqli", "union select", "' or 1=1"}, IntentSQLi, 0.9},
		{[]string{"xss", "cross site", "cross-site", "<script>", "javascript:"}, IntentXSS, 0.9},
		{[]string{"命令注入", "command injection", "rce", "remote code", "; ls", "; dir", "&& ls"}, IntentCommandInj, 0.85},
		{[]string{"路径遍历", "path traversal", "directory traversal", "../"}, IntentPathTravers, 0.85},
		{[]string{"ssrf", "server-side request forgery", "http://127.0.0.1", "http://localhost"}, IntentSSRF, 0.85},
		{[]string{"文件包含", "lfi", "rfi", "file inclusion", "php://filter"}, IntentFileIncl, 0.85},
		// CTF
		{[]string{"ctf", "capture the flag"}, IntentCTF, 0.8},
		// 综合渗透
		{[]string{"安全测试", "渗透测试", "渗透", "漏洞", "vulnerability"}, IntentPentest, 0.75},
		// 普通开发
		{[]string{"创建", "写", "实现", "网站", "网页", "函数", "代码", "html", "css", "javascript", "python", "go"}, IntentCode, 0.8},
		{[]string{"规划", "计划", "分解", "roadmap"}, IntentPlan, 0.8},
		{[]string{"审查", "review", "检查", "分析"}, IntentReview, 0.8},
		{[]string{"测试", "test", "用例"}, IntentTest, 0.8},
		{[]string{"调试", "debug", "错误", "bug"}, IntentDebug, 0.8},
		{[]string{"git", "commit", "push", "pull", "branch", "仓库", "版本"}, IntentGit, 0.8},
		{[]string{"评估", "反馈", "改进"}, IntentFeedback, 0.8},
	}

	for _, sk := range specificKeywords {
		for _, keyword := range sk.keywords {
			if strings.Contains(inputLower, keyword) {
				intent = sk.intent
				confidence = sk.conf
				goto FOUND // 找到后直接退出
			}
		}
	}

FOUND:
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
		IntentDebug, IntentGit, IntentFeedback,
		// 安全测试细分
		IntentSQLi, IntentXSS, IntentCommandInj, IntentPathTravers,
		IntentSSRF, IntentFileIncl, IntentPentest, IntentCTF,
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
		IntentCode:         "Coder",
		IntentPlan:         "Planner",
		IntentReview:       "Reviewer",
		IntentTest:         "Tester",
		IntentDebug:        "Debugger",
		IntentGit:          "Git",
		IntentFeedback:     "Feedback",
		// 安全测试细分
		IntentSQLi:         "SQLiAgent",
		IntentXSS:          "XSSAgent",
		IntentCommandInj:   "CommandInjectAgent",
		IntentPathTravers:  "PathTraversalAgent",
		IntentSSRF:         "SSRFAgent",
		IntentFileIncl:     "FileIncludeAgent",
		IntentPentest:      "Pentesting",
		IntentCTF:          "CTFExploration",
		IntentChat:         "Orchestrator",
		IntentUnknown:      "Generic", // 默认路由到 Generic
	}

	if agent, ok := mapping[intent]; ok {
		return agent
	}
	return "Generic" // 默认路由
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
		// 新增的安全测试细分 Agent
		"SQLiAgent": true, "XSSAgent": true, "CommandInjectAgent": true,
		"PathTraversalAgent": true, "SSRFAgent": true, "FileIncludeAgent": true,
		"CTFExploration": true, "Generic": true,
	}
	return validAgents[name]
}
