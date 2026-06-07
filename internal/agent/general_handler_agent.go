package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/mcp"
)

type GeneralHandlerAgent struct {
	name      string
	llmClient *llm.Client
	tools     []Tool
	mcpInst   *mcp.MCP
	maxSteps  int
}

type Tool struct {
	Name        string
	Description string
	Usage       string
}

type LLMResponse struct {
	Thought string
	Action  string
	Params  map[string]string
}

func NewGeneralHandlerAgent(llmClient *llm.Client) *GeneralHandlerAgent {
	tools := []Tool{
		{Name: "browser_navigate", Description: "Navigate to a URL", Usage: `browser_navigate("http://example.com")`},
		{Name: "browser_click", Description: "Click an element on the page", Usage: `browser_click("Login")`},
		{Name: "browser_type", Description: "Type text into an input", Usage: `browser_type("Username", "admin")`},
		{Name: "browser_snapshot", Description: "Get the current page snapshot", Usage: `browser_snapshot()`},
		{Name: "browser_evaluate", Description: "Run JavaScript in the browser", Usage: `browser_evaluate("document.title")`},
		{Name: "task_complete", Description: "Mark the task as complete with a final result", Usage: `task_complete("summary of findings")`},
	}
	return &GeneralHandlerAgent{
		name:      "GeneralHandler",
		llmClient: llmClient,
		tools:     tools,
		mcpInst:   mcp.GetInstance(),
		maxSteps:  15,
	}
}

func (a *GeneralHandlerAgent) Name() string {
	return a.name
}

func (a *GeneralHandlerAgent) Description() string {
	return "General-purpose agent. Uses an LLM-driven action loop to complete uncertain tasks."
}

func (a *GeneralHandlerAgent) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[GeneralHandler] Task: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))

	history := []string{}
	var finalResult string

	for step := 1; step <= a.maxSteps; step++ {
		fmt.Printf("\n[Step %d/%d]\n", step, a.maxSteps)
		snapshot, err := a.getBrowserSnapshot()
		if err != nil {
			snapshot = "[snapshot unavailable]"
		}

		prompt := a.buildPrompt(task, history, snapshot, step)
		llmResp, err := a.llmClient.Chat(prompt)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}

		parsed := a.parseLLMResponse(llmResp)
		history = append(history, fmt.Sprintf("[Step %d] Thought: %s", step, parsed.Thought))

		if parsed.Action == "task_complete" {
			finalResult = parsed.Params["result"]
			break
		}

		if parsed.Action != "" {
			result, execErr := a.executeAction(parsed.Action, parsed.Params)
			if execErr != nil {
				history = append(history, fmt.Sprintf("[Step %d] Failed: %v", step, execErr))
			} else {
				history = append(history, fmt.Sprintf("[Step %d] Result: %s", step, result))
			}
		}
		time.Sleep(1 * time.Second)
	}

	if finalResult == "" {
		finalResult = fmt.Sprintf("Task ran for %d steps without explicit completion.", a.maxSteps)
	}
	return finalResult, nil
}

func (a *GeneralHandlerAgent) buildPrompt(task string, history []string, snapshot string, step int) string {
	var sb strings.Builder
	sb.WriteString("You are a general-purpose security testing agent.\n\n")
	sb.WriteString("===== Task =====\n")
	sb.WriteString(task + "\n\n")
	sb.WriteString("===== Available Tools =====\n")
	for _, tool := range a.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n  Usage: %s\n\n", tool.Name, tool.Description, tool.Usage))
	}
	sb.WriteString("===== Output Format =====\n")
	sb.WriteString("Thought: [your reasoning]\n")
	sb.WriteString("Action: [tool_call(args)]\n\n")
	sb.WriteString(fmt.Sprintf("===== Step %d =====\n", step))
	sb.WriteString("===== Page Snapshot =====\n")
	sb.WriteString(snapshot + "\n\n")
	if len(history) > 0 {
		sb.WriteString("===== History =====\n")
		for i := len(history) - 4; i < len(history); i++ {
			if i < 0 {
				continue
			}
			sb.WriteString(history[i] + "\n")
		}
	}
	sb.WriteString("\nProvide your thought and next action.")
	return sb.String()
}

func (a *GeneralHandlerAgent) parseLLMResponse(resp string) LLMResponse {
	result := LLMResponse{Thought: resp, Action: "", Params: map[string]string{}}
	if m := regexp.MustCompile(`(?i)thought[:：]\s*(.+?)(?=\s*action|$)`).FindStringSubmatch(resp); len(m) > 1 {
		result.Thought = strings.TrimSpace(m[1])
	}
	patterns := []struct {
		pattern string
		action  string
		paramFn func(string) map[string]string
	}{
		{`(?i)browser_navigate\(\s*["']([^"']+)["']\s*\)`, "browser_navigate", func(s string) map[string]string { return map[string]string{"url": s} }},
		{`(?i)browser_click\(\s*["']([^"']+)["']\s*\)`, "browser_click", func(s string) map[string]string { return map[string]string{"name": s} }},
		{`(?i)browser_type\(\s*["']([^"']+)["']\s*,\s*["']([^"']+)["']\s*\)`, "browser_type", func(s string) map[string]string {
			m := regexp.MustCompile(`["']([^"']+)["']\s*,\s*["']([^"']+)["']`).FindStringSubmatch(s)
			if len(m) >= 3 {
				return map[string]string{"placeholder": m[1], "text": m[2]}
			}
			return nil
		}},
		{`(?i)browser_snapshot\(\s*\)`, "browser_snapshot", func(string) map[string]string { return nil }},
		{`(?i)task_complete\(\s*["']([^"']+)["']\s*\)`, "task_complete", func(s string) map[string]string { return map[string]string{"result": s} }},
	}
	for _, p := range patterns {
		if m := regexp.MustCompile(p.pattern).FindStringSubmatch(resp); len(m) > 0 {
			result.Action = p.action
			result.Params = p.paramFn(m[0])
			return result
		}
	}
	return result
}

func (a *GeneralHandlerAgent) getBrowserSnapshot() (string, error) {
	return "[No active page]", nil
}

func (a *GeneralHandlerAgent) executeAction(action string, params map[string]string) (string, error) {
	switch action {
	case "browser_navigate":
		if u, ok := params["url"]; ok {
			return fmt.Sprintf("Navigated to %s", u), nil
		}
	case "browser_click":
		if n, ok := params["name"]; ok {
			return fmt.Sprintf("Clicked %s", n), nil
		}
	case "browser_type":
		if p, ok := params["placeholder"]; ok {
			if t, ok := params["text"]; ok {
				return fmt.Sprintf("Typed '%s' into '%s'", t, p), nil
			}
		}
	case "browser_snapshot":
		return a.getBrowserSnapshot()
	case "task_complete":
		if r, ok := params["result"]; ok {
			return r, nil
		}
		return "done", nil
	}
	return "", fmt.Errorf("invalid action or missing params: %s", action)
}
