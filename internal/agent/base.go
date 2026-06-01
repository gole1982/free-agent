package agent

import (
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

type BaseAgent struct {
	name    string
	gateway llm.Gateway
	store   *memory.Store
	convID  int64
}

func NewBaseAgent(name string, gateway llm.Gateway, store *memory.Store) (*BaseAgent, error) {
	conv, err := store.CreateConversation(name + " conversation")
	if err != nil {
		return nil, err
	}

	return &BaseAgent{
		name:    name,
		gateway: gateway,
		store:   store,
		convID:  conv.ID,
	}, nil
}

func (a *BaseAgent) Think(prompt string) (string, error) {
	fullPrompt := a.buildPrompt(prompt)
	response, err := a.gateway.Chat(fullPrompt)
	if err != nil {
		return "", err
	}

	a.store.AddMessage(a.convID, "user", fullPrompt)
	a.store.AddMessage(a.convID, "assistant", response)

	return response, nil
}

func (a *BaseAgent) buildPrompt(userPrompt string) string {
	messages, _ := a.store.GetMessages(a.convID)
	var history string
	for _, m := range messages {
		history += m.Role + ": " + m.Content + "\n"
	}
	return history + "user: " + userPrompt
}
