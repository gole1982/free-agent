package agent

import (
	"context"
	"fmt"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type TaskPlannerAgent struct {
	gateway *llm.SimpleGateway
}

func NewTaskPlannerAgent(gateway *llm.SimpleGateway) *TaskPlannerAgent {
	return &TaskPlannerAgent{gateway: gateway}
}

func (a *TaskPlannerAgent) Name() string {
	return "TaskPlanner"
}

func (a *TaskPlannerAgent) Description() string {
	return "Task planning and decomposition. Breaks down complex tasks into actionable steps."
}

func (a *TaskPlannerAgent) Execute(ctx context.Context, input string) (string, error) {
	prompt := fmt.Sprintf(`
You are a task planner AI. Break down the following task into clear, actionable steps:

Task: %s

Please provide a detailed plan with numbered steps. Each step should be specific and actionable.
`, input)
	return a.gateway.Chat(prompt)
}
