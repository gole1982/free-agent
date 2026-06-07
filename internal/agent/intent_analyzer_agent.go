package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type IntentType string

const (
	IntentCode        IntentType = "CODE"
	IntentPlan        IntentType = "PLAN"
	IntentReview      IntentType = "REVIEW"
	IntentTest        IntentType = "TEST"
	IntentDebug       IntentType = "DEBUG"
	IntentGit         IntentType = "GIT"
	IntentFeedback    IntentType = "FEEDBACK"
	IntentSQLi        IntentType = "SQLI"
	IntentXSS         IntentType = "XSS"
	IntentCommandInj  IntentType = "CMDINJ"
	IntentPathTravers IntentType = "PATHTRAV"
	IntentSSRF        IntentType = "SSRF"
	IntentFileIncl    IntentType = "FILEINCL"
	IntentPentest     IntentType = "PENTEST"
	IntentCTF         IntentType = "CTF"
	IntentChat        IntentType = "CHAT"
	IntentProject     IntentType = "PROJECT"
	IntentExploration IntentType = "EXPLORATION"
	IntentUnknown     IntentType = "UNKNOWN"
)

type IntentResult struct {
	Intent     IntentType        `json:"intent"`
	Confidence float64           `json:"confidence"`
	AgentName  string            `json:"agent"`
	Parameters map[string]string `json:"parameters"`
	Summary    string            `json:"summary"`
	NeedPlan   bool              `json:"need_plan"`
	NeedReview bool              `json:"need_review"`
}

type IntentAnalyzerAgent struct {
	gateway *llm.SimpleGateway
}

func NewIntentAnalyzerAgent(gateway *llm.SimpleGateway) *IntentAnalyzerAgent {
	return &IntentAnalyzerAgent{gateway: gateway}
}

func (a *IntentAnalyzerAgent) Name() string {
	return "IntentAnalyzer"
}

func (a *IntentAnalyzerAgent) Description() string {
	return "Natural language understanding. Classifies user intent and selects a target agent."
}

const intentAnalyzerSystemPrompt = `
You are a Natural Language Understanding system. Classify the user intent.

## Supported Intents (pick ONE)
- CODE: writing code, implementing features
- PLAN: planning projects, creating roadmaps
- REVIEW: code review, quality analysis
- TEST: writing tests
- DEBUG: debugging, fixing errors
- GIT: git operations
- FEEDBACK: evaluating results
- SQLI: SQL Injection testing
- XSS: XSS testing
- CMDINJ: Command Injection testing
- PATHTRAV: Path Traversal testing
- SSRF: SSRF testing
- FILEINCL: File Inclusion testing
- PENTEST: comprehensive security testing
- CTF: CTF challenges
- CHAT: general conversation
- PROJECT: complex projects requiring multi-agent collaboration (need_plan=true)
- EXPLORATION: uncertain problems requiring exploration (need_plan=true)
- UNKNOWN: cannot determine

## Rules
1. RESPOND WITH ONLY JSON - no explanations, no markdown
2. confidence must be between 0.0 and 1.0
3. agent must be one of: CodeGenerator, TaskPlanner, CodeReviewer, TestEngineer, DebugAnalyst, GitOperator, FeedbackCollector, SQLInjectionScanner, XSSScanner, CommandInjectionScanner, PathTraversalScanner, SSRFScanner, FileIncludeScanner, SecurityAssessor, CTFSolver, SolutionExplorer, GeneralHandler, TaskCoordinator

## Output Format
{"intent":"INTENT","confidence":0.95,"agent":"AgentName","summary":"brief summary","need_plan":false,"need_review":true}
`

func (a *IntentAnalyzerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf("%s\n\n## User Input\n%s", intentAnalyzerSystemPrompt, input)
	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return a.fallbackIntentParse(input)
	}
	result, err := a.parseIntentResponse(response)
	if err != nil {
		return a.fallbackIntentParse(input)
	}
	return a.formatIntentResult(result)
}

func (a *IntentAnalyzerAgent) parseIntentResponse(response string) (*IntentResult, error) {
	jsonStr, err := extractJSONObject(response)
	if err != nil {
		return nil, fmt.Errorf("extract json: %w", err)
	}
	var result IntentResult
	result.Parameters = make(map[string]string)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse intent json: %w", err)
	}
	if !isValidIntent(result.Intent) {
		result.Intent = IntentUnknown
	}
	if result.AgentName == "" || !isValidAgentName(result.AgentName) {
		result.AgentName = intentToAgent(result.Intent)
	}
	return &result, nil
}

func (a *IntentAnalyzerAgent) fallbackIntentParse(input string) (string, error) {
	lower := strings.ToLower(input)
	rules := []struct {
		keywords []string
		intent   IntentType
		conf     float64
	}{
		{[]string{"sql注入", "sql injection", "sqli", "union select", "' or 1=1"}, IntentSQLi, 0.9},
		{[]string{"xss", "cross site", "<script>", "javascript:"}, IntentXSS, 0.9},
		{[]string{"命令注入", "command injection", "rce", "; ls", "; dir"}, IntentCommandInj, 0.85},
		{[]string{"路径遍历", "path traversal", "directory traversal", "../"}, IntentPathTravers, 0.85},
		{[]string{"ssrf", "http://127.0.0.1", "http://localhost"}, IntentSSRF, 0.85},
		{[]string{"文件包含", "lfi", "rfi", "file inclusion", "php://filter"}, IntentFileIncl, 0.85},
		{[]string{"ctf", "capture the flag"}, IntentCTF, 0.8},
		{[]string{"安全测试", "渗透测试", "渗透", "漏洞", "vulnerability"}, IntentPentest, 0.75},
		{[]string{"创建", "写", "实现", "网站", "网页", "代码", "html", "css", "javascript", "python", "go"}, IntentCode, 0.8},
		{[]string{"规划", "计划", "分解", "roadmap"}, IntentPlan, 0.8},
		{[]string{"审查", "review", "检查"}, IntentReview, 0.8},
		{[]string{"测试", "test", "用例"}, IntentTest, 0.8},
		{[]string{"调试", "debug", "错误", "bug"}, IntentDebug, 0.8},
		{[]string{"git", "commit", "push", "pull", "branch", "仓库", "版本"}, IntentGit, 0.8},
		{[]string{"评估", "反馈", "改进"}, IntentFeedback, 0.8},
	}

	intent := IntentUnknown
	conf := 0.5
found:
	for _, r := range rules {
		for _, kw := range r.keywords {
			if strings.Contains(lower, kw) {
				intent = r.intent
				conf = r.conf
				break found
			}
		}
	}

	result := IntentResult{
		Intent:     intent,
		Confidence: conf,
		AgentName:  intentToAgent(intent),
		Summary:    input,
		Parameters: make(map[string]string),
		NeedReview: intent == IntentCode,
	}
	return a.formatIntentResult(&result)
}

func (a *IntentAnalyzerAgent) formatIntentResult(result *IntentResult) (string, error) {
	out := fmt.Sprintf(`INTENT_RESULT:
{
  "intent": "%s",
  "confidence": %.2f,
  "agent": "%s",
  "summary": "%s",
  "need_plan": %v,
  "need_review": %v
}`,
		result.Intent, result.Confidence, result.AgentName,
		result.Summary, result.NeedPlan, result.NeedReview)
	if len(result.Parameters) > 0 {
		out += "\nPARAMETERS:"
		for k, v := range result.Parameters {
			out += fmt.Sprintf("\n  %s: %s", k, v)
		}
	}
	return out, nil
}

func isValidIntent(intent IntentType) bool {
	for _, v := range []IntentType{
		IntentCode, IntentPlan, IntentReview, IntentTest,
		IntentDebug, IntentGit, IntentFeedback,
		IntentSQLi, IntentXSS, IntentCommandInj, IntentPathTravers,
		IntentSSRF, IntentFileIncl, IntentPentest, IntentCTF,
		IntentChat, IntentUnknown,
	} {
		if intent == v {
			return true
		}
	}
	return false
}

func intentToAgent(intent IntentType) string {
	mapping := map[IntentType]string{
		IntentCode:         "CodeGenerator",
		IntentPlan:         "TaskPlanner",
		IntentReview:       "CodeReviewer",
		IntentTest:         "TestEngineer",
		IntentDebug:        "DebugAnalyst",
		IntentGit:          "GitOperator",
		IntentFeedback:     "FeedbackCollector",
		IntentSQLi:         "SQLInjectionScanner",
		IntentXSS:          "XSSScanner",
		IntentCommandInj:   "CommandInjectionScanner",
		IntentPathTravers:  "PathTraversalScanner",
		IntentSSRF:         "SSRFScanner",
		IntentFileIncl:     "FileIncludeScanner",
		IntentPentest:      "SecurityAssessor",
		IntentCTF:          "CTFSolver",
		IntentChat:         "TaskCoordinator",
		IntentUnknown:      "GeneralHandler",
	}
	if name, ok := mapping[intent]; ok {
		return name
	}
	return "GeneralHandler"
}

func parseIntentResult(response string) (*IntentResult, error) {
	jsonStr, err := extractJSONObject(response)
	if err != nil {
		return nil, fmt.Errorf("extract json: %w", err)
	}
	var result IntentResult
	result.Parameters = make(map[string]string)
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse intent json: %w", err)
	}
	if result.AgentName == "" || !isValidAgentName(result.AgentName) {
		result.AgentName = intentToAgent(result.Intent)
	}
	return &result, nil
}

func isValidAgentName(name string) bool {
	switch name {
	case "CodeGenerator", "TaskPlanner", "CodeReviewer",
		"TestEngineer", "DebugAnalyst", "GitOperator", "FeedbackCollector",
		"SecurityAssessor", "TaskCoordinator",
		"SQLInjectionScanner", "XSSScanner", "CommandInjectionScanner",
		"PathTraversalScanner", "SSRFScanner", "FileIncludeScanner",
		"CTFSolver", "GeneralHandler", "SolutionExplorer":
		return true
	}
	return false
}
