package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// ExecutorExecutionInfo Executor执行过程中生成的关键信息
type ExecutorExecutionInfo struct {
	AgentName           string
	Task                string
	Output              string
	ExecutionFlag       string // normal/warning/error/honeypot/deadloop
	Timestamp           time.Time
	IntentMatch         float64 // 与用户意图的匹配度
	SuspiciousIndicators []string // 可疑指标
}

// ObserverDecision Observer的判断结果
type ObserverDecision struct {
	ShouldStop       bool     // 是否应该停止Executor
	Reason           string   // 原因
	CorrectGuidance  string   // 正确指引（如果需要）
	IntentAlignment  float64  // 与用户意图的匹配度
	IsDeadLoop       bool     // 是否检测到死循环
	IsMalicious      bool     // 是否检测到恶意行为
	IsHoneypot       bool     // 是否检测到蜜罐
}

// ObserverAgent Control Agent - Monitors Executor execution and determines if it follows user intent
// Uses Observer pattern terminology, following software engineering standard naming
type ObserverAgent struct {
	name              string
	llmClient         *llm.Client
	executorInfoChan  chan ExecutorExecutionInfo
	decisionChan      chan ObserverDecision
	stopChan          chan struct{}
	mu                sync.Mutex
	executorInfos     []ExecutorExecutionInfo
	originalIntent    string // 用户原始意图
}

// NewObserverAgent 创建Observer Agent
func NewObserverAgent(llmClient *llm.Client) *ObserverAgent {
	return &ObserverAgent{
		name:              "Observer",
		llmClient:         llmClient,
		executorInfoChan:  make(chan ExecutorExecutionInfo, 100),
		decisionChan:      make(chan ObserverDecision, 10),
		stopChan:          make(chan struct{}, 1), // 改为有缓冲，避免阻塞
		executorInfos:     make([]ExecutorExecutionInfo, 0),
	}
}

// Name 实现Agent接口
func (o *ObserverAgent) Name() string {
	return o.name
}

// Description implements Agent interface
func (o *ObserverAgent) Description() string {
	return "Control Agent - Monitors Executor execution and determines if it follows user intent (Observer pattern)"
}

// SetOriginalIntent sets user's original intent (used to determine if Executor follows it)
func (o *ObserverAgent) SetOriginalIntent(intent string) {
	o.mu.Lock()
	o.originalIntent = intent
	o.mu.Unlock()
}

// ReceiveExecutorInfo receives Executor execution info (called by scheduler)
func (o *ObserverAgent) ReceiveExecutorInfo(info ExecutorExecutionInfo) {
	o.executorInfoChan <- info
}

// GetDecision gets Observer's decision (called by scheduler)
func (o *ObserverAgent) GetDecision() ObserverDecision {
	select {
	case decision := <-o.decisionChan:
		return decision
	case <-time.After(100 * time.Millisecond):
		return ObserverDecision{ShouldStop: false}
	}
}

// StopExecutor stops monitoring and summarizes info (called by scheduler)
func (o *ObserverAgent) StopExecutor() {
	o.stopChan <- struct{}{}
}

// GetSummary gets execution info summary (called by scheduler)
func (o *ObserverAgent) GetSummary() []ExecutorExecutionInfo {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.executorInfos
}

// Execute implements Agent interface - Observer main execution logic
func (o *ObserverAgent) Execute(ctx context.Context, task string) (string, error) {
	// Observer's execution is a continuous monitoring process
	// It continuously receives Executor info and makes decisions
	
	for {
		select {
		case <-ctx.Done():
			return "Observer cancelled", ctx.Err()
			
		case <-o.stopChan:
			// Executor exited, summarize info
			return o.generateSummary(), nil
			
		case info := <-o.executorInfoChan:
			// Received Executor info, make decision
			o.mu.Lock()
			o.executorInfos = append(o.executorInfos, info)
			o.mu.Unlock()
			
			// Use LLM to determine if following user intent
			decision := o.analyzeExecutorInfo(ctx, info)
			
			if decision.ShouldStop {
				// Send stop decision
				o.decisionChan <- decision
				return fmt.Sprintf("Anomaly detected: %s\nGuidance: %s", decision.Reason, decision.CorrectGuidance), nil
			}
		}
	}
}

// analyzeExecutorInfo analyzes Executor execution info (using LLM)
func (o *ObserverAgent) analyzeExecutorInfo(ctx context.Context, info ExecutorExecutionInfo) ObserverDecision {
	o.mu.Lock()
	intent := o.originalIntent
	o.mu.Unlock()
	
	// Build decision prompt
	prompt := fmt.Sprintf(`You are an execution monitoring Agent (Observer). Decide whether the Executor is still aligned with the user's intent.

User original intent: %s

Executor execution info:
- Agent Name: %s
- Task: %s
- Output Summary: %s
- Execution Flag: %s
- Suspicious Indicators: %v

Decide:
1. Does the Executor follow the user's original intent?
2. Are there signs of infinite loop (repeated output, no progress)?
3. Is there malicious behavior (dangerous operations)?
4. Was a honeypot triggered (response contains "caught", "trapped", etc.)?

Respond with ONLY a JSON object wrapped in a single code block. No prose, no markdown headings.

`+"```json"+`
{
  "should_stop": false,
  "reason": "explanation when should_stop is true, otherwise empty",
  "correct_guidance": "guidance to redirect the Executor, empty when no issue",
  "intent_alignment": 0.85,
  "is_dead_loop": false,
  "is_malicious": false,
  "is_honeypot": false
}
`+"```",
		intent,
		info.AgentName,
		info.Task,
		truncate(info.Output, 500),
		info.ExecutionFlag,
		info.SuspiciousIndicators)
	
	// 使用带超时的context调用LLM，支持中断
	llmCtx, llmCancel := context.WithTimeout(ctx, 30*time.Second)
	defer llmCancel()
	
	// 调用LLM进行判断（在独立goroutine中执行，可被中断）
	type llmResult struct {
		response string
		err      error
	}
	resultChan := make(chan llmResult, 1)
	
	go func() {
		response, err := o.llmClient.Chat(prompt)
		resultChan <- llmResult{response, err}
	}()
	
	// 等待LLM结果或停止信号（支持外部取消和内部超时）
	select {
	case <-ctx.Done():
		// 外部取消，使用后备分析
		return o.fallbackAnalysis(info)
	case <-llmCtx.Done():
		// LLM调用超时，使用后备分析
		return o.fallbackAnalysis(info)
	case result := <-resultChan:
		if result.err != nil {
			// LLM调用失败，使用硬编码规则作为后备
			return o.fallbackAnalysis(info)
		}
		// 解析LLM返回的JSON
		decision := o.parseDecision(result.response)
		return decision
	}
}

// fallbackAnalysis fallback analysis (hardcoded rules for LLM failures)
func (o *ObserverAgent) fallbackAnalysis(info ExecutorExecutionInfo) ObserverDecision {
	decision := ObserverDecision{
		ShouldStop:      false,
		IntentAlignment: 0.8,
	}
	
	// Simple hardcoded rules (fallback only)
	lowerOutput := strings.ToLower(info.Output)
	
	// Error detection - stop immediately
	if info.ExecutionFlag == "error" {
		decision.ShouldStop = true
		decision.Reason = "Executor execution failed with error flag"
		decision.CorrectGuidance = "Check error details and adjust approach"
		return decision
	}
	
	// Honeypot detection
	if strings.Contains(lowerOutput, "caught") {
		decision.ShouldStop = true
		decision.IsHoneypot = true
		decision.Reason = "Honeypot triggered"
		decision.CorrectGuidance = "Stop this test type, continue other tests"
		return decision
	}
	
	// Deadloop detection (check historical output)
	o.mu.Lock()
	infos := o.executorInfos
	o.mu.Unlock()
	
	if len(infos) >= 2 {
		last := infos[len(infos)-1].Output
		prev := infos[len(infos)-2].Output
		if last == prev && last == info.Output && last != "" {
			decision.ShouldStop = true
			decision.IsDeadLoop = true
			decision.Reason = "Repeated output detected, possible dead loop"
			decision.CorrectGuidance = "Try different test methods or parameters"
			return decision
		}
	}
	
	return decision
}

// parseDecision parses decision JSON returned by LLM
func (o *ObserverAgent) parseDecision(response string) ObserverDecision {
	decision, err := parseObserverDecision(response)
	if err != nil {
		fmt.Printf("[Observer] JSON parse failed (%v); using fallback\n", err)
		return ObserverDecision{ShouldStop: false, IntentAlignment: 0.8}
	}
	return decision
}

// generateSummary generates execution info summary
func (o *ObserverAgent) generateSummary() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	var summary strings.Builder
	summary.WriteString("=== Executor Execution Summary ===\n")
	
	for i, info := range o.executorInfos {
		summary.WriteString(fmt.Sprintf("\n[%d] Agent: %s\n", i+1, info.AgentName))
		summary.WriteString(fmt.Sprintf("    Task: %s\n", truncate(info.Task, 100)))
		summary.WriteString(fmt.Sprintf("    Flag: %s\n", info.ExecutionFlag))
		summary.WriteString(fmt.Sprintf("    Time: %s\n", info.Timestamp.Format("15:04:05")))
		if len(info.SuspiciousIndicators) > 0 {
			summary.WriteString(fmt.Sprintf("    Suspicious Indicators: %v\n", info.SuspiciousIndicators))
		}
	}
	
	return summary.String()
}