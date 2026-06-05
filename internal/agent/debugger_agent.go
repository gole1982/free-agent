package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type DebuggerAgent struct {
	gateway *llm.SimpleGateway
}

func NewDebuggerAgent(gateway *llm.SimpleGateway) *DebuggerAgent {
	return &DebuggerAgent{gateway: gateway}
}

func (a *DebuggerAgent) Name() string {
	return "Debugger"
}

func (a *DebuggerAgent) Description() string {
	return "Error analysis and debugging. Helps identify and fix bugs in code."
}

func (a *DebuggerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a debugger expert. Analyze the following error or problematic code:

Input:
%s

Please provide:
1. Root cause analysis
2. Step-by-step debugging approach
3. Fix suggestions
4. Code corrections if applicable

Be as detailed as possible.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
