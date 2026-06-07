package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type FeedbackCollectorAgent struct {
	gateway *llm.SimpleGateway
}

func NewFeedbackCollectorAgent(gateway *llm.SimpleGateway) *FeedbackCollectorAgent {
	return &FeedbackCollectorAgent{gateway: gateway}
}

func (a *FeedbackCollectorAgent) Name() string {
	return "FeedbackCollector"
}

func (a *FeedbackCollectorAgent) Description() string {
	return "Result evaluation and feedback. Analyzes output quality and provides improvement suggestions."
}

func (a *FeedbackCollectorAgent) Execute(ctx context.Context, input string) (string, error) {
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
`, input)
	return a.gateway.Chat(prompt)
}
