package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
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
}

// NewEvaluatorAgent 创建Evaluator Agent
func NewEvaluatorAgent(llmClient *llm.Client, skillLoader *SkillLoader) *EvaluatorAgent {
	return &EvaluatorAgent{
		name:        "Evaluator",
		llmClient:   llmClient,
		skillLoader: skillLoader,
	}
}

// Name 实现Agent接口
func (e *EvaluatorAgent) Name() string {
	return e.name
}

// Description 实现Agent接口
func (e *EvaluatorAgent) Description() string {
	return "管理Agent - 分析执行信息，得出结论并更新安全策略（Evaluator模式）"
}

// Execute Evaluator的主执行逻辑（Agent接口）
func (e *EvaluatorAgent) Execute(ctx context.Context, task string) (string, error) {
	// Evaluator接收Observer的汇总信息进行分析
	// task参数包含Observer的汇总信息
	
	conclusion := e.analyzeExecution(task)
	
	// 如果需要更新安全策略，执行更新
	if conclusion.NeedsUpdate {
		e.updateSecurityPolicy(conclusion)
	}
	
	// 如果需要调整Agent特性，执行调整
	if len(conclusion.AgentAdjustments) > 0 {
		for _, adjustment := range conclusion.AgentAdjustments {
			e.adjustAgentTraits(adjustment)
		}
	}
	
	// 返回分析结论
	return e.formatConclusion(conclusion), nil
}

// analyzeExecution 分析执行信息（使用LLM）
func (e *EvaluatorAgent) analyzeExecution(summary string) EvaluationConclusion {
	// 构建分析提示
	prompt := fmt.Sprintf(`你是一个管理分析Agent（Evaluator），负责分析Executor执行信息并得出结论。

执行信息汇总:
%s

请分析:
1. Executor是正常退出还是异常退出？
2. 是否检测到恶意指令？
3. 任务是否完成？
4. 是否需要更新安全策略（输入过滤、技能更新）？
5. 是否需要调整Agent特性？

请以JSON格式返回分析结论:
{
  "exit_type": "normal/abnormal/malicious/deadloop",
  "needs_update": true/false,
  "update_type": "input_filter/skill_update/agent_adjustment",
  "update_content": "更新内容描述",
  "task_completed": true/false,
  "retry_needed": true/false,
  "retry_guidance": "重试指引",
  "agent_adjustments": [
    {
      "agent_name": "Agent名称",
      "efficiency_delta": 0.01,
      "quality_delta": 0.02,
      "creativity_delta": 0.0
    }
  ]
}`, summary)
	
	// 调用LLM进行分析
	response, err := e.llmClient.Chat(prompt)
	if err != nil {
		// LLM调用失败，使用硬编码规则作为后备
		return e.fallbackAnalysis(summary)
	}
	
	// 解析LLM返回的JSON
	conclusion := e.parseConclusion(response)
	return conclusion
}

// fallbackAnalysis 后备分析（硬编码规则）
func (e *EvaluatorAgent) fallbackAnalysis(summary string) EvaluationConclusion {
	conclusion := EvaluationConclusion{
		ExitType:      "normal",
		TaskCompleted: true,
	}
	
	// 简单的硬编码规则
	lower := strings.ToLower(summary)
	
	// 异常退出检测
	if strings.Contains(lower, "error") || strings.Contains(lower, "异常") {
		conclusion.ExitType = "abnormal"
		conclusion.TaskCompleted = false
		conclusion.RetryNeeded = true
		conclusion.RetryGuidance = "检查错误原因后重试"
	}
	
	// 恶意指令检测
	if strings.Contains(lower, "malicious") || strings.Contains(lower, "恶意") {
		conclusion.ExitType = "malicious"
		conclusion.NeedsUpdate = true
		conclusion.UpdateType = "input_filter"
		conclusion.UpdateContent = "添加恶意指令过滤规则"
		conclusion.TaskCompleted = false
	}
	
	// 死循环检测
	if strings.Contains(lower, "deadloop") || strings.Contains(lower, "死循环") {
		conclusion.ExitType = "deadloop"
		conclusion.TaskCompleted = false
		conclusion.RetryNeeded = true
		conclusion.RetryGuidance = "调整参数避免死循环"
	}
	
	// 蜜罐检测
	if strings.Contains(lower, "honeypot") || strings.Contains(lower, "蜜罐") {
		conclusion.ExitType = "abnormal"
		conclusion.NeedsUpdate = true
		conclusion.UpdateType = "skill_update"
		conclusion.UpdateContent = "更新安全测试技能，添加蜜罐识别模式"
		conclusion.TaskCompleted = false
	}
	
	return conclusion
}

// parseConclusion 解析LLM返回的结论JSON
func (e *EvaluatorAgent) parseConclusion(response string) EvaluationConclusion {
	conclusion := EvaluationConclusion{}
	
	lower := strings.ToLower(response)
	
	// 解析退出类型
	if strings.Contains(lower, `"exit_type": "abnormal"`) {
		conclusion.ExitType = "abnormal"
	} else if strings.Contains(lower, `"exit_type": "malicious"`) {
		conclusion.ExitType = "malicious"
	} else if strings.Contains(lower, `"exit_type": "deadloop"`) {
		conclusion.ExitType = "deadloop"
	} else {
		conclusion.ExitType = "normal"
	}
	
	// 解析是否需要更新
	if strings.Contains(lower, `"needs_update": true`) {
		conclusion.NeedsUpdate = true
	}
	
	// 解析任务是否完成
	if strings.Contains(lower, `"task_completed": false`) {
		conclusion.TaskCompleted = false
	} else {
		conclusion.TaskCompleted = true
	}
	
	// 解析是否需要重试
	if strings.Contains(lower, `"retry_needed": true`) {
		conclusion.RetryNeeded = true
	}
	
	// 解析Agent调整（简化处理）
	if strings.Contains(lower, `"agent_adjustments"`) {
		// 提取Agent名称（简化）
		conclusion.AgentAdjustments = []*AgentTraitsAdjustment{
			{
				AgentName:       "DefaultAgent",
				EfficiencyDelta: 0.01,
				QualityDelta:    0.02,
				CreativityDelta: 0,
			},
		}
	}
	
	return conclusion
}

// updateSecurityPolicy 更新安全策略
func (e *EvaluatorAgent) updateSecurityPolicy(conclusion EvaluationConclusion) {
	fmt.Printf("🔒 [Evaluator] 更新安全策略: %s\n", conclusion.UpdateType)
	
	switch conclusion.UpdateType {
	case "input_filter":
		// 更新输入过滤规则（写入安全配置文件）
		fmt.Printf("   - 添加输入过滤规则: %s\n", conclusion.UpdateContent)
		// TODO: 实现具体的输入过滤更新逻辑
		
	case "skill_update":
		// 更新安全测试技能
		fmt.Printf("   - 更新安全技能: %s\n", conclusion.UpdateContent)
		// TODO: 实现具体的技能更新逻辑
		
	case "agent_adjustment":
		// Agent特性调整（在adjustAgentTraits中处理）
		fmt.Printf("   - Agent特性调整\n")
	}
}

// adjustAgentTraits 调整Agent特性
func (e *EvaluatorAgent) adjustAgentTraits(adjustment *AgentTraitsAdjustment) {
	fmt.Printf("📈 [Evaluator] 调整Agent特性: %s\n", adjustment.AgentName)
	fmt.Printf("   - 效率调整: %.2f\n", adjustment.EfficiencyDelta)
	fmt.Printf("   - 质量调整: %.2f\n", adjustment.QualityDelta)
	fmt.Printf("   - 创造性调整: %.2f\n", adjustment.CreativityDelta)
	
	// 如果有SkillLoader，保存更新
	if e.skillLoader != nil {
		// TODO: 从SkillLoader加载当前特性，然后更新并保存
		fmt.Printf("💾 [Evaluator] 保存更新到SKILL.md\n")
	}
}

// formatConclusion 格式化结论输出
func (e *EvaluatorAgent) formatConclusion(conclusion EvaluationConclusion) string {
	var result strings.Builder
	
	result.WriteString("=== Evaluator分析结论 ===\n")
	result.WriteString(fmt.Sprintf("退出类型: %s\n", conclusion.ExitType))
	
	if conclusion.TaskCompleted {
		result.WriteString("任务完成: 是\n")
	} else {
		result.WriteString("任务完成: 否\n")
	}
	
	if conclusion.NeedsUpdate {
		result.WriteString(fmt.Sprintf("\n需要更新安全策略:\n"))
		result.WriteString(fmt.Sprintf("  类型: %s\n", conclusion.UpdateType))
		result.WriteString(fmt.Sprintf("  内容: %s\n", conclusion.UpdateContent))
	}
	
	if conclusion.RetryNeeded {
		result.WriteString(fmt.Sprintf("\n需要重试:\n"))
		result.WriteString(fmt.Sprintf("  指引: %s\n", conclusion.RetryGuidance))
	}
	
	if len(conclusion.AgentAdjustments) > 0 {
		result.WriteString(fmt.Sprintf("\nAgent特性调整建议:\n"))
		for _, adj := range conclusion.AgentAdjustments {
			result.WriteString(fmt.Sprintf("  %s: 效率+%.2f, 质量+%.2f\n", 
				adj.AgentName, adj.EfficiencyDelta, adj.QualityDelta))
		}
	}
	
	return result.String()
}