package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type ExplorationAgent struct {
	gateway       *llm.SimpleGateway
	tree          *ExplorationTree
	agentMgr      *AgentManager
	decisionEngine *DecisionEngine
}

func NewExplorationAgent(gateway *llm.SimpleGateway, agentMgr *AgentManager) *ExplorationAgent {
	de := NewDecisionEngine()
	de.AddPattern(StrategyPattern{
		ID:          "xss-pattern",
		Pattern:     "XSS",
		Strategy:    "scanning",
		SuccessRate: 0.85,
		Uses:        15,
		RiskLevel:   RiskHigh,
	})
	de.AddPattern(StrategyPattern{
		ID:          "sqli-pattern",
		Pattern:     "SQL",
		Strategy:    "scanning",
		SuccessRate: 0.78,
		Uses:        12,
		RiskLevel:   RiskCritical,
	})
	de.AddPattern(StrategyPattern{
		ID:          "buffer-pattern",
		Pattern:     "buffer",
		Strategy:    "exploitation",
		SuccessRate: 0.65,
		Uses:        8,
		RiskLevel:   RiskCritical,
	})
	de.AddPattern(StrategyPattern{
		ID:          "analysis-pattern",
		Pattern:     "analyze",
		Strategy:    "analysis",
		SuccessRate: 0.95,
		Uses:        20,
		RiskLevel:   RiskLow,
	})

	return &ExplorationAgent{
		gateway:        gateway,
		agentMgr:       agentMgr,
		decisionEngine: de,
	}
}

func (a *ExplorationAgent) Name() string {
	return "Explorer"
}

func (a *ExplorationAgent) Description() string {
	return "Exploration agent. Uses exploration tree to handle uncertain problems with backtracking and multi-branch exploration."
}

func (a *ExplorationAgent) Execute(ctx context.Context, task string) (string, error) {
	a.tree = NewExplorationTree(task)
	fmt.Println("[Explorer] Starting exploration for task:", task)

	return a.explore(ctx, task)
}

func (a *ExplorationAgent) explore(ctx context.Context, task string) (string, error) {
	a.tree.UpdateCurrentNodeStatus(NodeRunning, "Starting exploration", nil)

	riskLevel := a.decisionEngine.EvaluateRisk(task)
	fmt.Printf("[Explorer] Task risk level evaluated: %v\n", riskLevel)

	suggestedStrategies := a.decisionEngine.SuggestStrategies(task, 5)
	fmt.Printf("[Explorer] Decision engine suggested strategies: %v\n", suggestedStrategies)

	bestStrategy := a.decisionEngine.SelectBestStrategy(suggestedStrategies, task)
	fmt.Printf("[Explorer] Selected best strategy: %s\n", bestStrategy)

	strategies := a.generateStrategiesWithPriority(suggestedStrategies, task)
	fmt.Printf("[Explorer] Generated %d prioritized strategies\n", len(strategies))

	attempts := 0

	for _, strategy := range strategies {
		if !a.decisionEngine.ShouldContinue(attempts, 0.5) {
			fmt.Printf("[Explorer] Decision engine recommends stopping after %d attempts\n", attempts)
			break
		}

		node := a.tree.CreateChildNode(strategy.Type, strategy.Description, strategy.Payload)
		a.tree.MoveToNode(node.ID)

		result, err := a.executeStrategy(ctx, strategy)
		attempts++
		
		if err == nil {
			a.tree.UpdateCurrentNodeStatus(NodeSuccess, result, nil)
			fmt.Printf("[Explorer] Strategy succeeded: %s\n", strategy.Description)
			a.decisionEngine.UpdatePatternSuccess(strategy.Type+"-pattern", true)
			
			if strategy.Terminal {
				finalResult := a.consolidateResult()
				a.tree.UpdateNodeStatus("root", NodeSuccess, finalResult, nil)
				return finalResult, nil
			}
		} else {
			a.tree.UpdateCurrentNodeStatus(NodeFailed, "", err)
			fmt.Printf("[Explorer] Strategy failed: %v\n", err)
			a.decisionEngine.UpdatePatternSuccess(strategy.Type+"-pattern", false)
		}

		if attempts < len(strategies) {
			fmt.Println("[Explorer] Backtracking to try next strategy")
			a.tree.Backtrack()
		}
	}

	a.tree.UpdateNodeStatus("root", NodeFailed, "", fmt.Errorf("all strategies failed after %d attempts", attempts))
	return "", fmt.Errorf("all exploration strategies failed after %d attempts", attempts)
}

func (a *ExplorationAgent) generateStrategiesWithPriority(strategyNames []string, task string) []ExplorationStrategy {
	var strategies []ExplorationStrategy
	risk := a.decisionEngine.EvaluateRisk(task)

	for _, name := range strategyNames {
		priority := a.decisionEngine.CalculatePriority(name, risk)
		
		terminal := false
		if name == "exploitation" || name == "verification" {
			terminal = true
		}

		strategy := ExplorationStrategy{
			Type:        name,
			Description: a.getStrategyDescription(name, task),
			Terminal:    terminal,
			Payload:     map[string]interface{}{"priority": priority, "risk": risk},
		}
		strategies = append(strategies, strategy)
	}

	for i := 0; i < len(strategies)-1; i++ {
		for j := i + 1; j < len(strategies); j++ {
			priorityI := strategies[i].Payload["priority"].(int)
			priorityJ := strategies[j].Payload["priority"].(int)
			if priorityJ > priorityI {
				strategies[i], strategies[j] = strategies[j], strategies[i]
			}
		}
	}

	return strategies
}

func (a *ExplorationAgent) getStrategyDescription(strategy, task string) string {
	switch strategy {
	case "analysis":
		return fmt.Sprintf("Analyze the problem: %s", task)
	case "scanning":
		return fmt.Sprintf("Scan for vulnerabilities related to: %s", task)
	case "exploitation":
		return fmt.Sprintf("Attempt exploitation for: %s", task)
	case "verification":
		return fmt.Sprintf("Verify findings for: %s", task)
	default:
		return fmt.Sprintf("Execute %s strategy for: %s", strategy, task)
	}
}

type ExplorationStrategy struct {
	Type        string
	Description string
	Payload     map[string]interface{}
	Terminal    bool
}

func (a *ExplorationAgent) generateInitialStrategies(ctx context.Context, task string) ([]ExplorationStrategy, error) {
	// 暂时使用预定义策略，LLM调用可用于后续扩展
	_ = fmt.Sprintf(`
Analyze this task and suggest 3-5 exploration strategies. For each strategy, provide:
1. Type (e.g., "scanning", "analysis", "exploitation")
2. Description (what the strategy does)
3. Whether this strategy is terminal (completes the task)

Task: %s

Format response as JSON array:
[
  {"type": "...", "description": "...", "terminal": false},
  ...
]
`, task)

	// 简化处理，创建一些策略
	strategies := []ExplorationStrategy{
		{
			Type:        "analysis",
			Description: "Analyze the problem and gather information",
			Terminal:    false,
		},
		{
			Type:        "scanning",
			Description: "Scan for potential vulnerabilities or issues",
			Terminal:    false,
		},
		{
			Type:        "exploitation",
			Description: "Attempt to exploit identified opportunities",
			Terminal:    true,
		},
		{
			Type:        "verification",
			Description: "Verify and validate findings",
			Terminal:    true,
		},
	}

	return strategies, nil
}

func (a *ExplorationAgent) executeStrategy(ctx context.Context, strategy ExplorationStrategy) (string, error) {
	var result string
	var err error

	switch strategy.Type {
	case "analysis":
		result, err = a.doAnalysis(ctx, strategy)
	case "scanning":
		result, err = a.doScanning(ctx, strategy)
	case "exploitation":
		result, err = a.doExploitation(ctx, strategy)
	case "verification":
		result, err = a.doVerification(ctx, strategy)
	default:
		err = fmt.Errorf("unknown strategy type: %s", strategy.Type)
	}

	return result, err
}

func (a *ExplorationAgent) doAnalysis(ctx context.Context, strategy ExplorationStrategy) (string, error) {
	agent, err := a.agentMgr.GetAgent("Planner")
	if err != nil {
		return "", err
	}
	return agent.Execute(ctx, fmt.Sprintf("Analyze: %s", strategy.Description))
}

func (a *ExplorationAgent) doScanning(ctx context.Context, strategy ExplorationStrategy) (string, error) {
	agent, err := a.agentMgr.GetAgent("Pentesting")
	if err != nil {
		return "", err
	}
	return agent.Execute(ctx, fmt.Sprintf("Scan: %s", strategy.Description))
}

func (a *ExplorationAgent) doExploitation(ctx context.Context, strategy ExplorationStrategy) (string, error) {
	agent, err := a.agentMgr.GetAgent("Coder")
	if err != nil {
		return "", err
	}
	return agent.Execute(ctx, fmt.Sprintf("Exploit: %s", strategy.Description))
}

func (a *ExplorationAgent) doVerification(ctx context.Context, strategy ExplorationStrategy) (string, error) {
	agent, err := a.agentMgr.GetAgent("Tester")
	if err != nil {
		return "", err
	}
	return agent.Execute(ctx, fmt.Sprintf("Verify: %s", strategy.Description))
}

func (a *ExplorationAgent) consolidateResult() string {
	successNodes := a.tree.GetSuccessNodes()
	
	var sb strings.Builder
	sb.WriteString("=== Exploration Complete ===\n\n")
	sb.WriteString("Successful strategies:\n")
	
	for _, node := range successNodes {
		sb.WriteString(fmt.Sprintf("  - [%s] %s\n", node.Strategy, node.Description))
		if node.Result != "" {
			sb.WriteString(fmt.Sprintf("    Result: %s\n", node.Result))
		}
	}

	sb.WriteString("\nExploration tree:\n")
	sb.WriteString(a.tree.PrintTree())

	return sb.String()
}