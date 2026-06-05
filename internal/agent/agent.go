package agent

import (
	"context"
	"fmt"
)

type Agent interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input string) (string, error)
}

type AgentManager struct {
	agents map[string]Agent
}

func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents: make(map[string]Agent),
	}
}

func (am *AgentManager) RegisterAgent(agent Agent) {
	am.agents[agent.Name()] = agent
}

func (am *AgentManager) GetAgent(name string) (Agent, error) {
	agent, exists := am.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}
	return agent, nil
}

func (am *AgentManager) ListAgents() []Agent {
	agents := make([]Agent, 0, len(am.agents))
	for _, agent := range am.agents {
		agents = append(agents, agent)
	}
	return agents
}

func (am *AgentManager) Execute(ctx context.Context, agentName, input string) (string, error) {
	agent, err := am.GetAgent(agentName)
	if err != nil {
		return "", err
	}
	return agent.Execute(ctx, input)
}
