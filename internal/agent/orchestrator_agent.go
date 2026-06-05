package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type OrchestratorAgent struct {
	gateway    *llm.SimpleGateway
	agentMgr   *AgentManager
	intentAgent *IntentAgent
}

func NewOrchestratorAgent(gateway *llm.SimpleGateway, agentMgr *AgentManager) *OrchestratorAgent {
	return &OrchestratorAgent{
		gateway:     gateway,
		agentMgr:    agentMgr,
		intentAgent: NewIntentAgent(gateway),
	}
}

func (a *OrchestratorAgent) Name() string {
	return "Orchestrator"
}

func (a *OrchestratorAgent) Description() string {
	return "Task orchestrator. Routes requests to appropriate agents based on intent analysis."
}

func (a *OrchestratorAgent) Execute(ctx context.Context, input string) (string, error) {
	// 第一步：使用Intent Agent进行意图识别
	intentResult, err := a.intentAgent.Execute(ctx, input)
	if err != nil {
		// 如果意图识别失败，回退到直接处理
		fmt.Printf("[Orchestrator] Intent analysis failed: %v, using direct handling\n", err)
		return a.handleDirectRequest(ctx, input)
	}

	// 解析意图结果
	intentInfo, err := GetIntentResult(intentResult)
	if err != nil {
		fmt.Printf("[Orchestrator] Failed to parse intent: %v\n", err)
		return a.handleDirectRequest(ctx, input)
	}

	fmt.Printf("[Orchestrator] Intent detected: %s (%.2f confidence), routing to %s\n", 
		intentInfo.Intent, intentInfo.Confidence, intentInfo.AgentName)

	// 检查置信度，如果太低则需要进一步处理
	if intentInfo.Confidence < 0.6 {
		fmt.Printf("[Orchestrator] Low confidence (%.2f), requesting clarification\n", intentInfo.Confidence)
		return fmt.Sprintf("我理解您的请求是：%s\n\n但我需要更多信息来准确处理。请确认或补充更多细节。", intentInfo.Summary), nil
	}

	// 第二步：根据意图路由到对应Agent
	selectedAgent := intentInfo.AgentName

	// 特殊处理：如果是闲聊，直接回答
	if string(intentInfo.Intent) == "CHAT" {
		return a.handleChat(ctx, input, intentInfo.Summary)
	}

	// 如果是未知意图，尝试直接处理
	if string(intentInfo.Intent) == "UNKNOWN" {
		return a.handleDirectRequest(ctx, input)
	}

	// 获取目标Agent
	agent, err := a.agentMgr.GetAgent(selectedAgent)
	if err != nil {
		fmt.Printf("[Orchestrator] Agent %s not found, using direct handling\n", selectedAgent)
		return a.handleDirectRequest(ctx, input)
	}

	fmt.Printf("[Orchestrator] Delegating to %s agent...\n", selectedAgent)

	// 第三步：执行Agent任务
	result, err := agent.Execute(ctx, input)
	if err != nil {
		return fmt.Sprintf("[%s] 执行失败: %v", selectedAgent, err), nil
	}

	// 第四步：如果需要审查，将结果交给Reviewer
	if intentInfo.NeedReview && string(intentInfo.Intent) == "CODE" {
		fmt.Printf("[Orchestrator] Code completed, sending to Reviewer for review...\n")
		reviewer, _ := a.agentMgr.GetAgent("Reviewer")
		if reviewer != nil {
			reviewResult, _ := reviewer.Execute(ctx, result)
			result = fmt.Sprintf("%s\n\n[Reviewer Feedback]\n%s", result, reviewResult)
		}
	}

	return result, nil
}

// handleDirectRequest 处理无法识别的请求
func (a *OrchestratorAgent) handleDirectRequest(ctx context.Context, input string) (string, error) {
	// 回退到传统的Agent选择方式
	agents := a.agentMgr.ListAgents()
	var agentList string
	for _, ag := range agents {
		if ag.Name() != "Orchestrator" {
			agentList += fmt.Sprintf("%s: %s\n", ag.Name(), ag.Description())
		}
	}

	prompt := fmt.Sprintf(`
Analyze the user request and select the most appropriate agent.

Available Agents:
%s

User Request: %s

Respond with ONLY the agent name.
`, agentList, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	selectedAgent := cleanAgentName(response)
	if selectedAgent == "" {
		// 完全没有识别出agent，直接用LLM回答
		return a.handleChat(ctx, input, "general request")
	}

	agent, err := a.agentMgr.GetAgent(selectedAgent)
	if err != nil {
		return a.handleChat(ctx, input, "general request")
	}

	result, err := agent.Execute(ctx, input)
	if err != nil {
		return fmt.Sprintf("执行失败: %v", err), nil
	}

	return result, nil
}

// handleChat 处理闲聊请求
func (a *OrchestratorAgent) handleChat(ctx context.Context, input string, topic string) (string, error) {
	prompt := fmt.Sprintf(`
You are a helpful AI assistant. Please answer the following question or request:

%s

Be concise and helpful. If this is a coding question, provide code examples. If this is a general question, give a clear answer.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

func cleanAgentName(name string) string {
	name = strings.TrimSpace(name)
	name = trimSpecialCharacters(name)
	return strings.TrimSpace(name)
}

func trimSpecialCharacters(s string) string {
	result := ""
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == ' ' {
			result += string(r)
		}
	}
	return result
}
