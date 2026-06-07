package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]ParameterSchema
	Execute(ctx context.Context, params map[string]any) (string, error)
}

type ParameterSchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name())
	}
	r.tools[tool.Name()] = tool
	return nil
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

func (r *ToolRegistry) GetSchemas() map[string]ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schemas := make(map[string]ToolSchema)
	for name, tool := range r.tools {
		schemas[name] = ToolSchema{
			Name:        name,
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		}
	}
	return schemas
}

type ToolSchema struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Parameters  map[string]ParameterSchema `json:"parameters"`
}

func (r *ToolRegistry) ToJSONSchema() (string, error) {
	schemas := r.GetSchemas()
	var result []ToolSchema
	for _, s := range schemas {
		result = append(result, s)
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

var GlobalRegistry = NewToolRegistry()

func RegisterTool(tool Tool) error {
	return GlobalRegistry.Register(tool)
}

func GetTool(name string) (Tool, bool) {
	return GlobalRegistry.Get(name)
}

func ListTools() []Tool {
	return GlobalRegistry.List()
}
