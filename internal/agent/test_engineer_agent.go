package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type TestEngineerAgent struct {
	gateway *llm.SimpleGateway
}

func NewTestEngineerAgent(gateway *llm.SimpleGateway) *TestEngineerAgent {
	return &TestEngineerAgent{gateway: gateway}
}

func (a *TestEngineerAgent) Name() string {
	return "TestEngineer"
}

func (a *TestEngineerAgent) Description() string {
	return "Test case generation. Creates comprehensive test suites for code validation."
}

func (a *TestEngineerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a QA engineer. Generate comprehensive test cases for the following code or requirements:

Input:
%s

Please provide:
1. Unit test cases
2. Integration test cases
3. Edge case scenarios
4. Expected results for each test
`, input)
	return a.gateway.Chat(prompt)
}
