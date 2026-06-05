package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type ReviewerAgent struct {
	gateway *llm.SimpleGateway
}

func NewReviewerAgent(gateway *llm.SimpleGateway) *ReviewerAgent {
	return &ReviewerAgent{gateway: gateway}
}

func (a *ReviewerAgent) Name() string {
	return "Reviewer"
}

func (a *ReviewerAgent) Description() string {
	return "Code review and quality analysis. Provides feedback on code quality, security, and best practices."
}

func (a *ReviewerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a senior code reviewer. Analyze the following code and provide detailed feedback:

Code:
%s

Please review for:
1. Code quality and readability
2. Potential bugs and edge cases
3. Security vulnerabilities
4. Performance optimizations
5. Best practices and improvements

Provide specific suggestions for improvement.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
