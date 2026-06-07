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

// ObserverAgent 控制Agent - 监控Executor执行，判断是否遵循用户意图
// 使用 Observer 模式术语，符合软件工程标准命名
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
		stopChan:          make(chan struct{}),
		executorInfos:     make([]ExecutorExecutionInfo, 0),
	}
}

// Name 实现Agent接口
func (o *ObserverAgent) Name() string {
	return o.name
}

// Description 实现Agent接口
func (o *ObserverAgent) Description() string {
	return "控制Agent - 监控Executor执行，判断是否遵循用户意图（Observer模式）"
}

// SetOriginalIntent 设置用户原始意图（用于判断Executor是否遵循）
func (o *ObserverAgent) SetOriginalIntent(intent string) {
	o.mu.Lock()
	o.originalIntent = intent
	o.mu.Unlock()
}

// ReceiveExecutorInfo 接收Executor执行信息（调度者调用）
func (o *ObserverAgent) ReceiveExecutorInfo(info ExecutorExecutionInfo) {
	o.executorInfoChan <- info
}

// GetDecision 获取Observer的判断结果（调度者调用）
func (o *ObserverAgent) GetDecision() ObserverDecision {
	select {
	case decision := <-o.decisionChan:
		return decision
	default:
		return ObserverDecision{ShouldStop: false}
	}
}

// StopExecutor 停止监控并汇总信息（调度者调用）
func (o *ObserverAgent) StopExecutor() {
	o.stopChan <- struct{}{}
}

// GetSummary 获取执行信息汇总（调度者调用）
func (o *ObserverAgent) GetSummary() []ExecutorExecutionInfo {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.executorInfos
}

// Execute Observer的主执行逻辑（Agent接口）
func (o *ObserverAgent) Execute(ctx context.Context, task string) (string, error) {
	// Observer的执行是持续监控过程
	// 它会持续接收Executor信息，并做出判断
	
	for {
		select {
		case <-ctx.Done():
			return "Observer被取消", ctx.Err()
			
		case <-o.stopChan:
			// Executor退出，汇总信息
			return o.generateSummary(), nil
			
		case info := <-o.executorInfoChan:
			// 接收到Executor信息，进行判断
			o.mu.Lock()
			o.executorInfos = append(o.executorInfos, info)
			o.mu.Unlock()
			
			// 使用LLM判断是否遵循用户意图
			decision := o.analyzeExecutorInfo(ctx, info)
			
			if decision.ShouldStop {
				// 发送停止决策
				o.decisionChan <- decision
				return fmt.Sprintf("检测到异常: %s\n指引: %s", decision.Reason, decision.CorrectGuidance), nil
			}
		}
	}
}

// analyzeExecutorInfo 分析Executor执行信息（使用LLM判断）
func (o *ObserverAgent) analyzeExecutorInfo(ctx context.Context, info ExecutorExecutionInfo) ObserverDecision {
	o.mu.Lock()
	intent := o.originalIntent
	o.mu.Unlock()
	
	// 构建判断提示
	prompt := fmt.Sprintf(`你是一个执行监控Agent（Observer），负责判断Executor是否遵循用户意图。

用户原始意图: %s

Executor执行信息:
- Agent名称: %s
- 任务: %s
- 输出摘要: %s
- 执行标记: %s
- 可疑指标: %v

请判断:
1. Executor是否遵循用户原始意图？
2. 是否有死循环迹象（重复输出、无进展）？
3. 是否有恶意行为（尝试执行危险操作）？
4. 是否触发蜜罐（响应中包含"被抓"、"caught"等）？

如果发现问题，请:
1. 说明原因
2. 给出正确指引

请以JSON格式返回判断结果:
{
  "should_stop": true/false,
  "reason": "原因说明",
  "correct_guidance": "正确指引",
  "intent_alignment": 0.0-1.0,
  "is_dead_loop": true/false,
  "is_malicious": true/false,
  "is_honeypot": true/false
}`, 
		intent,
		info.AgentName,
		info.Task,
		truncate(info.Output, 500),
		info.ExecutionFlag,
		info.SuspiciousIndicators)
	
	// 调用LLM进行判断
	response, err := o.llmClient.Chat(prompt)
	if err != nil {
		// LLM调用失败，使用硬编码规则作为后备
		return o.fallbackAnalysis(info)
	}
	
	// 解析LLM返回的JSON
	decision := o.parseDecision(response)
	return decision
}

// fallbackAnalysis 后备分析（硬编码规则，用于LLM失败时）
func (o *ObserverAgent) fallbackAnalysis(info ExecutorExecutionInfo) ObserverDecision {
	decision := ObserverDecision{
		ShouldStop:      false,
		IntentAlignment: 0.8,
	}
	
	// 简单的硬编码规则（仅作为后备）
	lowerOutput := strings.ToLower(info.Output)
	
	// 蜜罐检测
	if strings.Contains(lowerOutput, "被抓") || strings.Contains(lowerOutput, "caught") {
		decision.ShouldStop = true
		decision.IsHoneypot = true
		decision.Reason = "检测到蜜罐触发"
		decision.CorrectGuidance = "停止该类型测试，继续其他测试"
	}
	
	// 死循环检测（检查历史输出）
	o.mu.Lock()
	infos := o.executorInfos
	o.mu.Unlock()
	
	if len(infos) >= 3 {
		last := infos[len(infos)-1].Output
		prev := infos[len(infos)-2].Output
		if last == prev && last == info.Output && last != "" {
			decision.ShouldStop = true
			decision.IsDeadLoop = true
			decision.Reason = "检测到重复输出，可能死循环"
			decision.CorrectGuidance = "尝试不同的测试方法或参数"
		}
	}
	
	return decision
}

// parseDecision 解析LLM返回的决策JSON
func (o *ObserverAgent) parseDecision(response string) ObserverDecision {
	// 简化解析，提取关键信息
	decision := ObserverDecision{}
	
	lower := strings.ToLower(response)
	
	if strings.Contains(lower, `"should_stop": true`) {
		decision.ShouldStop = true
	}
	
	if strings.Contains(lower, `"is_dead_loop": true`) {
		decision.IsDeadLoop = true
	}
	
	if strings.Contains(lower, `"is_malicious": true`) {
		decision.IsMalicious = true
	}
	
	if strings.Contains(lower, `"is_honeypot": true`) {
		decision.IsHoneypot = true
	}
	
	// 提取原因和指引（简化处理）
	if idx := strings.Index(response, `"reason"`); idx >= 0 {
		start := strings.Index(response[idx:], `"`)
		if start >= 0 {
			end := strings.Index(response[idx+start+1:], `"`)
			if end >= 0 {
				decision.Reason = response[idx+start+1 : idx+start+1+end]
			}
		}
	}
	
	return decision
}

// generateSummary 生成执行信息汇总
func (o *ObserverAgent) generateSummary() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	
	var summary strings.Builder
	summary.WriteString("=== Executor执行信息汇总 ===\n")
	
	for i, info := range o.executorInfos {
		summary.WriteString(fmt.Sprintf("\n[%d] Agent: %s\n", i+1, info.AgentName))
		summary.WriteString(fmt.Sprintf("    任务: %s\n", truncate(info.Task, 100)))
		summary.WriteString(fmt.Sprintf("    标记: %s\n", info.ExecutionFlag))
		summary.WriteString(fmt.Sprintf("    时间: %s\n", info.Timestamp.Format("15:04:05")))
		if len(info.SuspiciousIndicators) > 0 {
			summary.WriteString(fmt.Sprintf("    可疑指标: %v\n", info.SuspiciousIndicators))
		}
	}
	
	return summary.String()
}