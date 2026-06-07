package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

// EvaluationConclusion Evaluator的分析结论
type EvaluationConclusion struct {
	ExitType          string   // normal/abnormal/malicious/deadloop
	NeedsUpdate       bool     // 是否需要更新安全策略
	UpdateType        string   // input_filter/skill_update/agent_adjustment
	UpdateContent     string   // 更新内容描述
	TaskCompleted     bool     // 任务是否完成
	RetryNeeded       bool     // 是否需要重试
	RetryGuidance     string   // 重试指引
	AgentAdjustments  []*AgentTraitsAdjustment // Agent特性调整建议
}

// EvaluatorAgent 管理Agent - 分析执行信息，得出结论并更新安全策略
// 使用 Evaluator 术语，符合评估系统标准命名
type EvaluatorAgent struct {
	name        string
	llmClient   *llm.Client
	skillLoader *SkillLoader
	store       *memory.Store
}

// NewEvaluatorAgent 创建Evaluator Agent
func NewEvaluatorAgent(llmClient *llm.Client, skillLoader *SkillLoader, store *memory.Store) *EvaluatorAgent {
	return &EvaluatorAgent{
		name:        "Evaluator",
		llmClient:   llmClient,
		skillLoader: skillLoader,
		store:       store,
	}
}

// Name 实现Agent接口
func (e *EvaluatorAgent) Name() string {
	return e.name
}

// Description implements Agent interface
func (e *EvaluatorAgent) Description() string {
	return "Management Agent - Analyzes execution information, draws conclusions, and updates security policies (Evaluator pattern)"
}

// Execute implements Agent interface - Evaluator main execution logic
func (e *EvaluatorAgent) Execute(ctx context.Context, task string) (string, error) {
	// Evaluator receives summary from Observer for analysis
	// task parameter contains Observer's summary information
	
	conclusion := e.analyzeExecution(task)
	
	// Update security policy if needed
	if conclusion.NeedsUpdate {
		e.updateSecurityPolicy(conclusion)
	}
	
	// Adjust agent traits if needed
	if len(conclusion.AgentAdjustments) > 0 {
		for _, adjustment := range conclusion.AgentAdjustments {
			e.adjustAgentTraits(adjustment)
		}
	}
	
	// Return analysis conclusion
	return e.formatConclusion(conclusion), nil
}

// analyzeExecution analyzes execution information (using LLM)
func (e *EvaluatorAgent) analyzeExecution(summary string) EvaluationConclusion {
	prompt := fmt.Sprintf(`You are a management analysis Agent (Evaluator). Analyze the Executor execution summary and produce a structured conclusion.

Execution Summary:
%s

Determine:
1. Did the Executor exit normally or abnormally?
2. Was any malicious instruction detected?
3. Was the task completed?
4. Is security policy update needed (input filter, skill update, agent adjustment)?
5. Are agent trait adjustments needed?

Respond with ONLY a JSON object wrapped in a single code block. No prose, no markdown headings.

`+"```json"+`
{
  "exit_type": "normal",
  "needs_update": false,
  "update_type": "input_filter",
  "update_content": "description of the update",
  "task_completed": true,
  "retry_needed": false,
  "retry_guidance": "guidance for retry, empty when not needed",
  "agent_adjustments": [
    {
      "agent_name": "AgentName",
      "efficiency_delta": 0.01,
      "quality_delta": 0.02,
      "creativity_delta": 0.0
    }
  ]
}
`+"```", summary)

	response, err := e.llmClient.Chat(prompt)
	if err != nil {
		return e.fallbackAnalysis(summary)
	}
	conclusion, err := parseEvaluationConclusion(response)
	if err != nil {
		fmt.Printf("[Evaluator] JSON parse failed (%v); using fallback\n", err)
		return e.fallbackAnalysis(summary)
	}
	return conclusion
}

// fallbackAnalysis fallback analysis (hardcoded rules)
func (e *EvaluatorAgent) fallbackAnalysis(summary string) EvaluationConclusion {
	conclusion := EvaluationConclusion{
		ExitType:      "normal",
		TaskCompleted: true,
	}
	
	// Simple hardcoded rules
	lower := strings.ToLower(summary)
	
	// Abnormal exit detection
	if strings.Contains(lower, "error") || strings.Contains(lower, "abnormal") {
		conclusion.ExitType = "abnormal"
		conclusion.TaskCompleted = false
		conclusion.RetryNeeded = true
		conclusion.RetryGuidance = "Check error cause and retry"
	}
	
	// Malicious instruction detection
	if strings.Contains(lower, "malicious") {
		conclusion.ExitType = "malicious"
		conclusion.NeedsUpdate = true
		conclusion.UpdateType = "input_filter"
		conclusion.UpdateContent = "Add malicious instruction filter rules"
		conclusion.TaskCompleted = false
	}
	
	// Deadloop detection
	if strings.Contains(lower, "deadloop") {
		conclusion.ExitType = "deadloop"
		conclusion.TaskCompleted = false
		conclusion.RetryNeeded = true
		conclusion.RetryGuidance = "Adjust parameters to avoid deadloop"
	}
	
	// Honeypot detection
	if strings.Contains(lower, "honeypot") {
		conclusion.ExitType = "abnormal"
		conclusion.NeedsUpdate = true
		conclusion.UpdateType = "skill_update"
		conclusion.UpdateContent = "Update security testing skills, add honeypot detection patterns"
		conclusion.TaskCompleted = false
	}
	
	return conclusion
}

// parseConclusion parses conclusion JSON returned by LLM (now via shared helper)
func (e *EvaluatorAgent) parseConclusion(response string) EvaluationConclusion {
	conclusion, err := parseEvaluationConclusion(response)
	if err != nil {
		return e.fallbackAnalysis(response)
	}
	return conclusion
}

// updateSecurityPolicy updates security policy and persists to store
func (e *EvaluatorAgent) updateSecurityPolicy(conclusion EvaluationConclusion) {
	fmt.Printf("[Evaluator] Updating security policy: %s\n", conclusion.UpdateType)

	if e.store == nil {
		fmt.Printf("[Evaluator] Warning: no store available, skipping persistence\n")
		return
	}

	switch conclusion.UpdateType {
	case "input_filter":
		existing, _ := e.store.GetSecurityPolicy("input_filter")
		if existing != nil {
			err := e.store.UpdateSecurityPolicy("input_filter", existing.PolicyContent+"\n"+conclusion.UpdateContent, true)
			if err != nil {
				fmt.Printf("[Evaluator] Failed to update input_filter: %v\n", err)
			}
		} else {
			err := e.store.SaveSecurityPolicy("input_filter", conclusion.UpdateContent, true)
			if err != nil {
				fmt.Printf("[Evaluator] Failed to save input_filter: %v\n", err)
			}
		}

	case "skill_update":
		existing, _ := e.store.GetSecurityPolicy("skill_update")
		if existing != nil {
			err := e.store.UpdateSecurityPolicy("skill_update", existing.PolicyContent+"\n"+conclusion.UpdateContent, true)
			if err != nil {
				fmt.Printf("[Evaluator] Failed to update skill_update: %v\n", err)
			}
		} else {
			err := e.store.SaveSecurityPolicy("skill_update", conclusion.UpdateContent, true)
			if err != nil {
				fmt.Printf("[Evaluator] Failed to save skill_update: %v\n", err)
			}
		}

	case "agent_adjustment":
		fmt.Printf("   - Agent traits adjustment (handled in adjustAgentTraits)\n")
	}
}

// adjustAgentTraits adjusts agent traits and persists to store
func (e *EvaluatorAgent) adjustAgentTraits(adjustment *AgentTraitsAdjustment) {
	fmt.Printf("[Evaluator] Adjusting agent traits: %s\n", adjustment.AgentName)
	fmt.Printf("   - Efficiency delta: %.2f\n", adjustment.EfficiencyDelta)
	fmt.Printf("   - Quality delta: %.2f\n", adjustment.QualityDelta)
	fmt.Printf("   - Creativity delta: %.2f\n", adjustment.CreativityDelta)

	if e.store == nil {
		fmt.Printf("[Evaluator] Warning: no store available, skipping persistence\n")
		return
	}

	existing, err := e.store.GetAgentTraits(adjustment.AgentName)
	if err != nil {
		fmt.Printf("[Evaluator] Failed to get agent traits: %v\n", err)
		return
	}

	existing.Efficiency = clamp(existing.Efficiency+adjustment.EfficiencyDelta, 0.0, 1.0)
	existing.Quality = clamp(existing.Quality+adjustment.QualityDelta, 0.0, 1.0)
	existing.Creativity = clamp(existing.Creativity+adjustment.CreativityDelta, 0.0, 1.0)

	err = e.store.SaveAgentTraits(existing)
	if err != nil {
		fmt.Printf("[Evaluator] Failed to save agent traits: %v\n", err)
	} else {
		fmt.Printf("[Evaluator] Saved traits for %s: E=%.2f Q=%.2f C=%.2f\n",
			adjustment.AgentName, existing.Efficiency, existing.Quality, existing.Creativity)
	}
}

// formatConclusion formats conclusion output
func (e *EvaluatorAgent) formatConclusion(conclusion EvaluationConclusion) string {
	var result strings.Builder
	
	result.WriteString("=== Evaluator Analysis Conclusion ===\n")
	result.WriteString(fmt.Sprintf("Exit Type: %s\n", conclusion.ExitType))
	
	if conclusion.TaskCompleted {
		result.WriteString("Task Completed: Yes\n")
	} else {
		result.WriteString("Task Completed: No\n")
	}
	
	if conclusion.NeedsUpdate {
		result.WriteString(fmt.Sprintf("\nSecurity Policy Update Required:\n"))
		result.WriteString(fmt.Sprintf("  Type: %s\n", conclusion.UpdateType))
		result.WriteString(fmt.Sprintf("  Content: %s\n", conclusion.UpdateContent))
	}
	
	if conclusion.RetryNeeded {
		result.WriteString(fmt.Sprintf("\nRetry Required:\n"))
		result.WriteString(fmt.Sprintf("  Guidance: %s\n", conclusion.RetryGuidance))
	}
	
	if len(conclusion.AgentAdjustments) > 0 {
		result.WriteString(fmt.Sprintf("\nAgent Traits Adjustments:\n"))
		for _, adj := range conclusion.AgentAdjustments {
			result.WriteString(fmt.Sprintf("  %s: Efficiency+%.2f, Quality+%.2f\n", 
				adj.AgentName, adj.EfficiencyDelta, adj.QualityDelta))
		}
	}
	
	return result.String()
}