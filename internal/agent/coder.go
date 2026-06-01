package agent

import (
	"fmt"
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

type CoderAgent struct {
	base *BaseAgent
}

func NewCoderAgent(gateway llm.Gateway, store *memory.Store) (*CoderAgent, error) {
	base, err := NewBaseAgent("Coder", gateway, store)
	if err != nil {
		return nil, err
	}
	return &CoderAgent{base: base}, nil
}

func (c *CoderAgent) WriteCode(task string) (string, error) {
	prompt := fmt.Sprintf(`You are a professional software engineer. Please write high-quality code for the following task.

Task: %s

Please provide:
1. The complete code
2. Explanation of how it works
3. Usage examples

Use markdown for code blocks.`, task)

	return c.base.Think(prompt)
}
