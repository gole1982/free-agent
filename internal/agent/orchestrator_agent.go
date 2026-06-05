package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type OrchestratorAgent struct {
	gateway    *llm.SimpleGateway
	agentMgr   *AgentManager
	intentAgent *IntentAgent
	agentPool  *AgentPool
}

func NewOrchestratorAgent(gateway *llm.SimpleGateway, agentMgr *AgentManager) *OrchestratorAgent {
	return &OrchestratorAgent{
		gateway:     gateway,
		agentMgr:    agentMgr,
		intentAgent: NewIntentAgent(gateway),
		agentPool:   NewAgentPool(agentMgr),
	}
}

func (a *OrchestratorAgent) Name() string {
	return "Orchestrator"
}

func (a *OrchestratorAgent) Description() string {
	return "Task orchestrator. Routes requests to appropriate agents. Supports multi-agent collaboration."
}

func (a *OrchestratorAgent) Execute(ctx context.Context, input string) (string, error) {
	intentResult, err := a.intentAgent.Execute(ctx, input)
	if err != nil {
		fmt.Printf("[Orchestrator] Intent analysis failed: %v, using direct handling\n", err)
		return a.handleDirectRequest(ctx, input)
	}

	intentInfo, err := GetIntentResult(intentResult)
	if err != nil {
		fmt.Printf("[Orchestrator] Failed to parse intent: %v\n", err)
		return a.handleDirectRequest(ctx, input)
	}

	fmt.Printf("[Orchestrator] Intent detected: %s (%.2f confidence), routing to %s\n",
		intentInfo.Intent, intentInfo.Confidence, intentInfo.AgentName)

	// 低置信度时直接使用 Generic Agent（像路由的默认路径）
	if intentInfo.Confidence < 0.6 {
		fmt.Printf("[Orchestrator] Low confidence (%.2f), falling back to Generic Agent\n", intentInfo.Confidence)
		genericAgent, err := a.agentMgr.GetAgent("Generic Agent")
		if err == nil {
			return genericAgent.Execute(ctx, input)
		}
		// 如果 Generic Agent 也不行，再尝试其他方式
		return fmt.Sprintf("我理解您的请求是：%s\n\n正在使用通用方法处理...", intentInfo.Summary), nil
	}

	if string(intentInfo.Intent) == "CHAT" {
		return a.handleChat(ctx, input, intentInfo.Summary)
	}

	if string(intentInfo.Intent) == "PROJECT" {
		return a.executeMultiAgentTask(ctx, input)
	}

	if string(intentInfo.Intent) == "EXPLORATION" {
		return a.executeExplorationTask(ctx, input)
	}

	if string(intentInfo.Intent) == "UNKNOWN" {
		return a.handleDirectRequest(ctx, input)
	}

	selectedAgent := intentInfo.AgentName
	agent, err := a.agentMgr.GetAgent(selectedAgent)
	if err != nil {
		fmt.Printf("[Orchestrator] Agent %s not found, using direct handling\n", selectedAgent)
		return a.handleDirectRequest(ctx, input)
	}

	fmt.Printf("[Orchestrator] Delegating to %s agent...\n", selectedAgent)
	result, err := agent.Execute(ctx, input)
	if err != nil {
		return fmt.Sprintf("[%s] 执行失败: %v", selectedAgent, err), nil
	}

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

func (a *OrchestratorAgent) executeMultiAgentTask(ctx context.Context, task string) (string, error) {
	fmt.Println("[Orchestrator] Executing multi-agent collaboration...")

	planner, err := a.agentMgr.GetAgent("Planner")
	if err != nil {
		return fmt.Sprintf("Planner agent not found: %v", err), nil
	}

	planResult, err := planner.Execute(ctx, fmt.Sprintf("Break down this task into independent subtasks. For each subtask, specify which agent type should handle it (Coder, Tester, Debugger, Reviewer). Format as numbered list: %s", task))
	if err != nil {
		return fmt.Sprintf("Planning failed: %v", err), nil
	}

	fmt.Printf("[Orchestrator] Plan received:\n%s\n", planResult)

	subtasks := a.parsePlanToTasks(planResult, task)
	fmt.Printf("[Orchestrator] Extracted %d subtasks\n", len(subtasks))

	if len(subtasks) == 0 {
		return "Failed to extract subtasks from plan", nil
	}

	results, err := a.executeSubtasks(ctx, subtasks)
	if err != nil {
		return fmt.Sprintf("Execution failed: %v", err), nil
	}

	finalResult := a.consolidateResults(subtasks, results)
	return finalResult, nil
}

func (a *OrchestratorAgent) executeExplorationTask(ctx context.Context, task string) (string, error) {
	fmt.Println("[Orchestrator] Executing exploration task (uncertainty handling)...")

	explorer, err := a.agentMgr.GetAgent("Explorer")
	if err != nil {
		return fmt.Sprintf("Explorer agent not found: %v", err), nil
	}

	result, err := explorer.Execute(ctx, task)
	if err != nil {
		return fmt.Sprintf("Exploration failed: %v", err), nil
	}

	return result, nil
}

func (a *OrchestratorAgent) parsePlanToTasks(plan string, parentTask string) []AgentTask {
	var tasks []AgentTask
	lines := strings.Split(plan, "\n")
	taskID := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			content := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			task := a.analyzeAndCreateTask(content, parentTask, taskID)
			if task.ID != "" {
				tasks = append(tasks, task)
				taskID++
			}
		} else if strings.Contains(line, ". ") && len(line) < 200 {
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				task := a.analyzeAndCreateTask(parts[1], parentTask, taskID)
				if task.ID != "" {
					tasks = append(tasks, task)
					taskID++
				}
			}
		}
	}

	return tasks
}

func (a *OrchestratorAgent) analyzeAndCreateTask(content, parentTask string, id int) AgentTask {
	task := AgentTask{
		ID:        fmt.Sprintf("%s-%d", parentTask[:min(len(parentTask), 20)], id),
		Input:     content,
		Context:   context.Background(),
		Priority:  1,
		RetryCount: 0,
	}

	lowerContent := strings.ToLower(content)

	if strings.Contains(lowerContent, "code") || strings.Contains(lowerContent, "write") ||
		strings.Contains(lowerContent, "implement") || strings.Contains(lowerContent, "create") ||
		strings.Contains(lowerContent, "build") {
		task.AgentType = "Coder"
	} else if strings.Contains(lowerContent, "test") || strings.Contains(lowerContent, "unit") {
		task.AgentType = "Tester"
	} else if strings.Contains(lowerContent, "debug") || strings.Contains(lowerContent, "fix") {
		task.AgentType = "Debugger"
	} else if strings.Contains(lowerContent, "review") || strings.Contains(lowerContent, "quality") {
		task.AgentType = "Reviewer"
	} else if strings.Contains(lowerContent, "plan") || strings.Contains(lowerContent, "design") {
		task.AgentType = "Planner"
	} else {
		task.AgentType = "Coder"
	}

	return task
}

func (a *OrchestratorAgent) executeSubtasks(ctx context.Context, tasks []AgentTask) ([]AgentResult, error) {
	tasksByType := make(map[string][]AgentTask)
	for _, task := range tasks {
		tasksByType[task.AgentType] = append(tasksByType[task.AgentType], task)
	}

	var allResults []AgentResult
	var wg sync.WaitGroup
	resultChan := make(chan []AgentResult, len(tasksByType))

	for agentType, typeTasks := range tasksByType {
		wg.Add(1)
		go func(at string, ts []AgentTask) {
			defer wg.Done()
			fmt.Printf("[Orchestrator] Executing %d %s tasks in parallel\n", len(ts), at)
			a.agentPool.AddWorkers(at, len(ts))
			results := a.agentPool.ExecuteParallel(ts)
			resultChan <- results
		}(agentType, typeTasks)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for results := range resultChan {
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func (a *OrchestratorAgent) consolidateResults(tasks []AgentTask, results []AgentResult) string {
	var builder strings.Builder

	builder.WriteString("=== Multi-Agent Collaboration Complete ===\n\n")
	builder.WriteString("Task Summary:\n")
	builder.WriteString("=============\n\n")

	for i, task := range tasks {
		builder.WriteString(fmt.Sprintf("Task %d (%s): %s\n", i+1, task.AgentType, task.Input))
	}

	builder.WriteString("\nResults:\n")
	builder.WriteString("========\n\n")

	agentStats := make(map[string]int)
	for _, result := range results {
		agentStats[result.AgentType]++
		if result.Error != nil {
			builder.WriteString(fmt.Sprintf("[%s] Failed: %v\n", result.AgentID, result.Error))
		} else {
			builder.WriteString(fmt.Sprintf("[%s]\n%s\n\n", result.AgentID, result.Output))
		}
	}

	builder.WriteString("=== Collaboration Report ===\n")
	builder.WriteString(fmt.Sprintf("Total subtasks: %d\n", len(tasks)))
	builder.WriteString("Agents involved:\n")
	for agentType, count := range agentStats {
		builder.WriteString(fmt.Sprintf("  - %s: %d tasks\n", agentType, count))
	}

	return builder.String()
}

func (a *OrchestratorAgent) handleDirectRequest(ctx context.Context, input string) (string, error) {
	agents := a.agentMgr.ListAgents()
	var agentList string
	for _, ag := range agents {
		if ag.Name() != "Orchestrator" {
			agentList += fmt.Sprintf("%s: %s\n", ag.Name(), ag.Description())
		}
	}

	// 先尝试找专门的Agent
	prompt := fmt.Sprintf(`
分析用户请求，选择最合适的Agent。

可用Agent列表:
%s

用户请求: %s

请判断：
1. 如果这个请求是明确的编程、测试、调试、代码审查等任务，选择对应的专门Agent
2. 如果这个请求是安全测试、渗透测试、CTF任务但不确定具体用什么专门Agent，选择"Generic Agent"
3. 如果完全不确定，或者是需要通用探索的任务，选择"Generic Agent"

请只返回Agent名称。
`, agentList, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	selectedAgent := cleanAgentName(response)
	
	// 如果没有选择到专门Agent，或者选择了Generic Agent，就用Generic Agent
	if selectedAgent == "" || strings.Contains(strings.ToLower(selectedAgent), "generic") {
		fmt.Println("[Orchestrator] 路由到 Generic Agent (不确定任务类型)...")
		agent, err := a.agentMgr.GetAgent("Generic Agent")
		if err == nil {
			result, execErr := agent.Execute(ctx, input)
			if execErr == nil {
				return result, nil
			}
		}
		// 如果Generic Agent也失败，用聊天模式
		return a.handleChat(ctx, input, "general request")
	}

	fmt.Printf("[Orchestrator] 路由到 %s...\n", selectedAgent)
	agent, err := a.agentMgr.GetAgent(selectedAgent)
	if err != nil {
		fmt.Printf("[Orchestrator] %s 不存在，尝试 Generic Agent...\n", selectedAgent)
		genericAgent, gErr := a.agentMgr.GetAgent("Generic Agent")
		if gErr == nil {
			return genericAgent.Execute(ctx, input)
		}
		return a.handleChat(ctx, input, "general request")
	}

	result, err := agent.Execute(ctx, input)
	if err != nil {
		return fmt.Sprintf("执行失败: %v", err), nil
	}

	return result, nil
}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}