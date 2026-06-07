package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type SolutionExplorerAgent struct {
	gateway  *llm.SimpleGateway
	tree     *ExplorationTree
	agentMgr *AgentManager
}

func NewSolutionExplorerAgent(gateway *llm.SimpleGateway, agentMgr *AgentManager) *SolutionExplorerAgent {
	return &SolutionExplorerAgent{
		gateway:  gateway,
		agentMgr: agentMgr,
	}
}

func (a *SolutionExplorerAgent) Name() string {
	return "SolutionExplorer"
}

func (a *SolutionExplorerAgent) Description() string {
	return "Exploration agent. Uses a tree to handle uncertain problems with backtracking and multi-branch exploration."
}

func (a *SolutionExplorerAgent) Execute(ctx context.Context, task string) (string, error) {
	a.tree = NewExplorationTree(task)
	fmt.Println("[SolutionExplorer] Starting exploration for task:", task)
	return a.explore(ctx, task)
}

func (a *SolutionExplorerAgent) explore(ctx context.Context, task string) (string, error) {
	a.tree.UpdateCurrentNodeStatus(NodeRunning, "Starting exploration", nil)

	strategies := a.defaultStrategies()
	fmt.Printf("[SolutionExplorer] Generated %d strategies\n", len(strategies))

	attempts := 0
	for _, strategy := range strategies {
		if attempts >= 5 {
			break
		}
		node := a.tree.CreateChildNode(strategy.Type, strategy.Description, strategy.Payload)
		a.tree.MoveToNode(node.ID)

		result, err := a.executeStrategy(ctx, strategy)
		attempts++

		if err == nil {
			a.tree.UpdateCurrentNodeStatus(NodeSuccess, result, nil)
			fmt.Printf("[SolutionExplorer] Strategy succeeded: %s\n", strategy.Description)
			if strategy.Terminal {
				finalResult := a.consolidateResult()
				a.tree.UpdateNodeStatus("root", NodeSuccess, finalResult, nil)
				return finalResult, nil
			}
		} else {
			a.tree.UpdateCurrentNodeStatus(NodeFailed, "", err)
			fmt.Printf("[SolutionExplorer] Strategy failed: %v\n", err)
		}
	}

	a.tree.UpdateNodeStatus("root", NodeFailed, "", fmt.Errorf("all strategies failed after %d attempts", attempts))
	return a.consolidateResult(), nil
}

type explorationStrategyItem struct {
	Type        string
	Description string
	Payload     map[string]interface{}
	Terminal    bool
}

func (a *SolutionExplorerAgent) defaultStrategies() []explorationStrategyItem {
	return []explorationStrategyItem{
		{Type: "analysis", Description: "Analyze the problem and gather information", Terminal: false},
		{Type: "scanning", Description: "Scan for potential vulnerabilities or issues", Terminal: false},
		{Type: "exploitation", Description: "Attempt to exploit identified opportunities", Terminal: true},
		{Type: "verification", Description: "Verify and validate findings", Terminal: true},
	}
}

func (a *SolutionExplorerAgent) executeStrategy(ctx context.Context, strategy explorationStrategyItem) (string, error) {
	agentName := ""
	switch strategy.Type {
	case "analysis":
		agentName = "TaskPlanner"
	case "scanning":
		agentName = "SecurityAssessor"
	case "exploitation":
		agentName = "CodeGenerator"
	case "verification":
		agentName = "TestEngineer"
	default:
		return "", fmt.Errorf("unknown strategy: %s", strategy.Type)
	}
	ag, err := a.agentMgr.GetAgent(agentName)
	if err != nil {
		return "", err
	}
	return ag.Execute(ctx, fmt.Sprintf("%s: %s", strategy.Type, strategy.Description))
}

func (a *SolutionExplorerAgent) consolidateResult() string {
	successNodes := a.tree.GetSuccessNodes()
	var b strings.Builder
	b.WriteString("=== Exploration Complete ===\n\n")
	b.WriteString("Successful strategies:\n")
	for _, node := range successNodes {
		b.WriteString(fmt.Sprintf("  - [%s] %s\n", node.Strategy, node.Description))
		if node.Result != "" {
			b.WriteString(fmt.Sprintf("    Result: %s\n", node.Result))
		}
	}
	b.WriteString("\nExploration tree:\n")
	b.WriteString(a.tree.PrintTree())
	return b.String()
}
