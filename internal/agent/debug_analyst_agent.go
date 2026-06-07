package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type DebugAnalystAgent struct {
	gateway *llm.SimpleGateway
}

func NewDebugAnalystAgent(gateway *llm.SimpleGateway) *DebugAnalystAgent {
	return &DebugAnalystAgent{gateway: gateway}
}

func (a *DebugAnalystAgent) Name() string {
	return "DebugAnalyst"
}

func (a *DebugAnalystAgent) Description() string {
	return "Error analysis and debugging. Helps identify and fix bugs in code."
}

func (a *DebugAnalystAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a debugger expert. Analyze the following error or problematic code:

Input:
%s

Please provide:
1. Root cause analysis
2. Step-by-step debugging approach
3. Fix suggestions
4. Code corrections if applicable
`, input)
	return a.gateway.Chat(prompt)
}
