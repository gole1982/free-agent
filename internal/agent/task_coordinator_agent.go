package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type TaskCoordinatorAgent struct {
	gateway     *llm.SimpleGateway
	agentMgr    *AgentManager
	intentAgent *IntentAnalyzerAgent
}

func NewTaskCoordinatorAgent(gateway *llm.SimpleGateway, agentMgr *AgentManager) *TaskCoordinatorAgent {
	return &TaskCoordinatorAgent{
		gateway:     gateway,
		agentMgr:    agentMgr,
		intentAgent: NewIntentAnalyzerAgent(gateway),
	}
}

func (a *TaskCoordinatorAgent) Name() string {
	return "TaskCoordinator"
}

func (a *TaskCoordinatorAgent) Description() string {
	return "Task coordinator. Routes requests to the appropriate agent based on intent analysis."
}

func (a *TaskCoordinatorAgent) Execute(ctx context.Context, input string) (string, error) {
	intentResult, err := a.intentAgent.Execute(ctx, input)
	if err != nil {
		return a.handleDirectRequest(ctx, input)
	}

	intentInfo, err := parseIntentResult(intentResult)
	if err != nil {
		return a.handleDirectRequest(ctx, input)
	}

	fmt.Printf("[TaskCoordinator] Intent: %s (confidence=%.2f) -> %s\n",
		intentInfo.Intent, intentInfo.Confidence, intentInfo.AgentName)

	if intentInfo.Confidence < 0.6 {
		return a.delegateTo(ctx, "GeneralHandler", input)
	}

	if intentInfo.NeedPlan {
		return a.executeMultiAgentTask(ctx, input)
	}

	return a.delegateTo(ctx, intentInfo.AgentName, input)
}

func (a *TaskCoordinatorAgent) executeMultiAgentTask(ctx context.Context, task string) (string, error) {
	planner, err := a.agentMgr.GetAgent("TaskPlanner")
	if err != nil {
		return a.handleDirectRequest(ctx, task)
	}

	planResult, err := planner.Execute(ctx, fmt.Sprintf("Break down this task into independent subtasks. Specify which agent type should handle each subtask: %s", task))
	if err != nil {
		return a.handleDirectRequest(ctx, task)
	}

	subtasks := parsePlanToTasks(planResult)
	if len(subtasks) == 0 {
		return a.handleDirectRequest(ctx, task)
	}

	var b strings.Builder
	b.WriteString("=== Multi-Agent Plan ===\n")
	b.WriteString(fmt.Sprintf("Plan:\n%s\n\n", planResult))
	b.WriteString("=== Subtask Results ===\n")

	for i, sub := range subtasks {
		out, err := a.delegateToSilent(ctx, sub.AgentType, sub.Input)
		if err != nil {
			b.WriteString(fmt.Sprintf("\n[%d] %s -> error: %v\n", i+1, sub.AgentType, err))
			continue
		}
		b.WriteString(fmt.Sprintf("\n[%d] %s\n%s\n", i+1, sub.AgentType, out))
	}
	return b.String(), nil
}

func (a *TaskCoordinatorAgent) delegateTo(ctx context.Context, agentName, input string) (string, error) {
	out, err := a.delegateToSilent(ctx, agentName, input)
	if err != nil {
		return "", err
	}
	fmt.Printf("[TaskCoordinator] %s completed\n", agentName)
	return out, nil
}

func (a *TaskCoordinatorAgent) delegateToSilent(ctx context.Context, agentName, input string) (string, error) {
	if agentName == "" {
		agentName = "GeneralHandler"
	}
	ag, err := a.agentMgr.GetAgent(agentName)
	if err != nil {
		ag, err = a.agentMgr.GetAgent("GeneralHandler")
		if err != nil {
			return "", fmt.Errorf("no fallback agent available: %w", err)
		}
	}
	return ag.Execute(ctx, input)
}

func (a *TaskCoordinatorAgent) handleDirectRequest(ctx context.Context, input string) (string, error) {
	return a.delegateTo(ctx, "GeneralHandler", input)
}

type subtaskItem struct {
	AgentType string
	Input     string
}

func parsePlanToTasks(plan string) []subtaskItem {
	var out []subtaskItem
	for _, line := range strings.Split(plan, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var content string
		switch {
		case strings.HasPrefix(line, "- "):
			content = strings.TrimPrefix(line, "- ")
		case strings.HasPrefix(line, "* "):
			content = strings.TrimPrefix(line, "* ")
		default:
			if idx := strings.Index(line, ". "); idx > 0 && idx < 4 {
				content = line[idx+2:]
			} else {
				continue
			}
		}
		out = append(out, subtaskItem{AgentType: classifyTask(content), Input: content})
	}
	return out
}

func classifyTask(content string) string {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "test") || strings.Contains(lower, "unit"):
		return "TestEngineer"
	case strings.Contains(lower, "debug") || strings.Contains(lower, "fix"):
		return "DebugAnalyst"
	case strings.Contains(lower, "review") || strings.Contains(lower, "quality"):
		return "CodeReviewer"
	case strings.Contains(lower, "plan") || strings.Contains(lower, "design"):
		return "TaskPlanner"
	default:
		return "CodeGenerator"
	}
}
