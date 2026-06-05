package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type FeedbackAgent struct {
	gateway *llm.SimpleGateway
}

func NewFeedbackAgent(gateway *llm.SimpleGateway) *FeedbackAgent {
	return &FeedbackAgent{gateway: gateway}
}

func (a *FeedbackAgent) Name() string {
	return "Feedback"
}

func (a *FeedbackAgent) Description() string {
	return "Result evaluation and feedback. Analyzes output quality and provides improvement suggestions."
}

func (a *FeedbackAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a feedback and evaluation AI. Analyze the following output and provide constructive feedback:

Output to evaluate:
%s

Please provide:
1. Quality assessment (1-5 stars)
2. Strengths of the output
3. Areas for improvement
4. Specific suggestions for revision
5. Overall recommendation

Be objective and constructive.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
