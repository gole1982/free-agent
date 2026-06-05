package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type TesterAgent struct {
	gateway *llm.SimpleGateway
}

func NewTesterAgent(gateway *llm.SimpleGateway) *TesterAgent {
	return &TesterAgent{gateway: gateway}
}

func (a *TesterAgent) Name() string {
	return "Tester"
}

func (a *TesterAgent) Description() string {
	return "Test case generation and execution. Creates comprehensive test suites for code validation."
}

func (a *TesterAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a QA engineer. Generate comprehensive test cases for the following code or requirements:

Input:
%s

Please provide:
1. Unit test cases
2. Integration test cases
3. Edge case scenarios
4. Expected results for each test

Include test code where applicable.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
