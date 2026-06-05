package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type PlannerAgent struct {
	gateway *llm.SimpleGateway
}

func NewPlannerAgent(gateway *llm.SimpleGateway) *PlannerAgent {
	return &PlannerAgent{gateway: gateway}
}

func (a *PlannerAgent) Name() string {
	return "Planner"
}

func (a *PlannerAgent) Description() string {
	return "Task planning and decomposition. Breaks down complex tasks into actionable steps."
}

func (a *PlannerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a task planner AI. Break down the following task into clear, actionable steps:

Task: %s

Please provide a detailed plan with numbered steps. Each step should be specific and actionable.
Include estimates for each step if applicable.
`, input)

	response, err := a.gateway.Chat(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}
