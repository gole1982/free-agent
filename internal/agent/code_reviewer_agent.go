package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type CodeReviewerAgent struct {
	gateway *llm.SimpleGateway
}

func NewCodeReviewerAgent(gateway *llm.SimpleGateway) *CodeReviewerAgent {
	return &CodeReviewerAgent{gateway: gateway}
}

func (a *CodeReviewerAgent) Name() string {
	return "CodeReviewer"
}

func (a *CodeReviewerAgent) Description() string {
	return "Code review and quality analysis. Provides feedback on code quality, security, and best practices."
}

func (a *CodeReviewerAgent) Execute(ctx context.Context, input string) (string, error) {
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
`, input)
	return a.gateway.Chat(prompt)
}
