package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Tool struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []Param  `json:"parameters"`
	Function    func(map[string]interface{}) (string, error)
}

type Param struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
}

type MCP struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

var instance *MCP
var once sync.Once

func GetInstance() *MCP {
	once.Do(func() {
		instance = &MCP{
			tools: make(map[string]*Tool),
		}
	})
	return instance
}

func (m *MCP) RegisterTool(tool *Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	m.tools[tool.Name] = tool
	return nil
}

func (m *MCP) GetTool(name string) (*Tool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, exists := m.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool, nil
}

func (m *MCP) ListTools() []*Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]*Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}

	return tools
}

func (m *MCP) ExecuteTool(name string, params map[string]interface{}) (string, error) {
	tool, err := m.GetTool(name)
	if err != nil {
		return "", err
	}

	return tool.Function(params)
}

func (m *MCP) GetToolJSON(name string) (string, error) {
	tool, err := m.GetTool(name)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(tool)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (m *MCP) ListToolsJSON() (string, error) {
	tools := m.ListTools()
	
	result := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		})
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
